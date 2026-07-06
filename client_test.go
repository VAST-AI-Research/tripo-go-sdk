package tripo3d

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(ClientOptions{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Timeout: 5 * time.Second,
		Retries: 1,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return server, client
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSONBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var m map[string]interface{}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal body: %v (body=%s)", err, data)
		}
	}
	return m
}

func TestNewClientRequiresAPIKey(t *testing.T) {
	saved, had := os.LookupEnv("TRIPO_API_KEY")
	os.Unsetenv("TRIPO_API_KEY")
	defer func() {
		if had {
			os.Setenv("TRIPO_API_KEY", saved)
		}
	}()

	_, err := NewClient(ClientOptions{})
	if err == nil {
		t.Fatal("expected an error when no API key is available")
	}
}

func TestTextToModelPostsExpectedPayload(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generation/text-to-model" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected Authorization header: %s", got)
		}
		body := readJSONBody(t, r)
		if body["prompt"] != "a cat" {
			t.Fatalf("unexpected prompt: %v", body["prompt"])
		}
		if _, ok := body["negative_prompt"]; ok {
			t.Fatalf("negative_prompt should have been omitted, got %v", body)
		}
		writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]string{"task_id": "task_123"}})
	})

	id, err := client.TextToModel(context.Background(), TextToModelParams{
		Prompt:  "a cat",
		Model:   String(ModelVersionH31),
		Texture: Bool(true),
	})
	if err != nil {
		t.Fatalf("TextToModel: %v", err)
	}
	if id != "task_123" {
		t.Fatalf("expected task_123, got %s", id)
	}
}

func TestTextToModelRejectsEmptyPrompt(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	})
	_, err := client.TextToModel(context.Background(), TextToModelParams{Prompt: "  "})
	if err == nil {
		t.Fatal("expected an error for an empty prompt")
	}
}

func TestImageToModelAcceptsAURL(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := readJSONBody(t, r)
		file, ok := body["file"].(map[string]interface{})
		if !ok || file["url"] != "https://ex.com/a.png" {
			t.Fatalf("unexpected file field: %v", body["file"])
		}
		writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]string{"task_id": "task_img"}})
	})

	id, err := client.ImageToModel(context.Background(), ImageToModelParams{
		File: File("https://ex.com/a.png"),
	})
	if err != nil {
		t.Fatalf("ImageToModel: %v", err)
	}
	if id != "task_img" {
		t.Fatalf("expected task_img, got %s", id)
	}
}

func TestImageToModelRejectsMissingFile(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	})
	_, err := client.ImageToModel(context.Background(), ImageToModelParams{})
	if err == nil {
		t.Fatal("expected an error when File is empty")
	}
}

func TestMultiviewToModelRequiresFilesOrOriginalTaskID(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	})
	_, err := client.MultiviewToModel(context.Background(), MultiviewToModelParams{})
	if err == nil {
		t.Fatal("expected an error when neither Files nor OriginalTaskID is set")
	}
}

func TestWaitForTaskPollsUntilSuccess(t *testing.T) {
	var calls int32
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		status := "running"
		if n >= 3 {
			status = "success"
		}
		writeJSON(w, 200, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"task_id": "task_xyz", "type": "text_to_model", "status": status, "progress": n * 30,
			},
		})
	})

	var seen []TaskStatus
	task, err := client.WaitForTask(context.Background(), "task_xyz", WaitOptions{
		PollInterval: time.Millisecond,
		OnProgress:   func(t *Task) { seen = append(seen, t.Status) },
	})
	if err != nil {
		t.Fatalf("WaitForTask: %v", err)
	}
	if task.Status != TaskStatusSuccess {
		t.Fatalf("expected success, got %s", task.Status)
	}
	if len(seen) == 0 || seen[len(seen)-1] != TaskStatusSuccess {
		t.Fatalf("expected last observed status to be success, got %v", seen)
	}
}

func TestWaitForTaskReturnsTaskErrorOnFailure(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"task_id": "task_fail", "type": "text_to_model", "status": "failed",
				"error_code": 42, "error_msg": "nope",
			},
		})
	})

	_, err := client.WaitForTask(context.Background(), "task_fail", WaitOptions{PollInterval: time.Millisecond})
	var taskErr *TaskError
	if !asTaskError(err, &taskErr) {
		t.Fatalf("expected *TaskError, got %T: %v", err, err)
	}
	if taskErr.Task.ErrorCode != 42 {
		t.Fatalf("expected error_code=42, got %d", taskErr.Task.ErrorCode)
	}
}

func TestWaitForTaskRespectsTimeout(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{
			"code": 0,
			"data": map[string]interface{}{
				"task_id": "task_slow", "type": "text_to_model", "status": "running", "progress": 5,
			},
		})
	})

	_, err := client.WaitForTask(context.Background(), "task_slow", WaitOptions{
		PollInterval: 5 * time.Millisecond,
		Timeout:      20 * time.Millisecond,
	})
	var timeoutErr *TimeoutError
	if !asTimeoutError(err, &timeoutErr) {
		t.Fatalf("expected *TimeoutError, got %T: %v", err, err)
	}
}

func TestAPIErrorResponseRaisesAPIError(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{
			"code": 2010, "message": "Insufficient credits", "suggestion": "Please top up your account",
		})
	})

	_, err := client.TextToModel(context.Background(), TextToModelParams{Prompt: "a cat"})
	var apiErr *APIError
	if !asAPIError(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.Code != 2010 || apiErr.Message != "Insufficient credits" {
		t.Fatalf("unexpected APIError: %+v", apiErr)
	}
}

func TestRigCheckRigModelAndRetargetBuildCorrectPayloads(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		body := readJSONBody(t, r)
		switch r.URL.Path {
		case "/animations/rig-check":
			if body["input"] != "task_src" {
				t.Fatalf("unexpected rig-check body: %v", body)
			}
			writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]string{"task_id": "chk_1"}})
		case "/animations/rig":
			if body["input"] != "task_src" || body["rig_type"] != "biped" || body["spec"] != "mixamo" {
				t.Fatalf("unexpected rig body: %v", body)
			}
			writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]string{"task_id": "rig_1"}})
		case "/animations/retarget":
			if body["input"] != "rig_1" {
				t.Fatalf("unexpected retarget body: %v", body)
			}
			writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]string{"task_id": "anim_1"}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	ctx := context.Background()
	chk, err := client.RigCheck(ctx, RigCheckParams{Input: "task_src"})
	if err != nil || chk != "chk_1" {
		t.Fatalf("RigCheck: id=%s err=%v", chk, err)
	}
	rig, err := client.RigModel(ctx, RigModelParams{Input: "task_src", RigType: String("biped"), Spec: String("mixamo")})
	if err != nil || rig != "rig_1" {
		t.Fatalf("RigModel: id=%s err=%v", rig, err)
	}
	anim, err := client.RetargetAnimation(ctx, RetargetAnimationParams{
		Input:      rig,
		Animations: []string{"preset:walk", "preset:idle"},
	})
	if err != nil || anim != "anim_1" {
		t.Fatalf("RetargetAnimation: id=%s err=%v", anim, err)
	}
}

func TestRetargetAnimationRejectsMoreThanFiveAnimations(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called")
	})
	animations := make([]string, 6)
	for i := range animations {
		animations[i] = "preset:walk"
	}
	_, err := client.RetargetAnimation(context.Background(), RetargetAnimationParams{
		Input:      "x",
		Animations: animations,
	})
	if err == nil {
		t.Fatal("expected an error for more than 5 animations")
	}
}

func TestGetBalanceParsesTheStandardEnvelope(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]float64{"balance": 12.5, "frozen": 0}})
	})

	balance, err := client.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if balance.Balance != 12.5 {
		t.Fatalf("expected balance 12.5, got %v", balance.Balance)
	}
}

func TestUploadFileReturnsFileToken(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		if header.Filename != "a.png" {
			t.Fatalf("unexpected filename: %s", header.Filename)
		}
		writeJSON(w, 200, map[string]interface{}{"code": 0, "data": map[string]string{"file_token": "ftok-123"}})
	})

	uploaded, err := client.UploadFile(context.Background(), []byte{1, 2, 3}, "a.png", "image/png")
	if err != nil {
		t.Fatalf("UploadFile: %v", err)
	}
	if uploaded.FileToken != "ftok-123" {
		t.Fatalf("expected ftok-123, got %s", uploaded.FileToken)
	}
}

func TestDownloadModelFetchesThePrimaryModelURL(t *testing.T) {
	server, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blob/model.glb" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "model/gltf-binary")
		w.WriteHeader(200)
		_, _ = w.Write([]byte{0x67, 0x6c, 0x54, 0x46})
	})

	task := &Task{
		TaskID: "t1", Type: "text_to_model", Status: TaskStatusSuccess,
		Output: &TaskOutput{ModelURL: fmt.Sprintf("%s/blob/model.glb", server.URL)},
	}
	downloaded, err := client.DownloadModel(context.Background(), task)
	if err != nil {
		t.Fatalf("DownloadModel: %v", err)
	}
	if downloaded == nil {
		t.Fatal("expected a non-nil download result")
	}
	if downloaded.ContentType != "model/gltf-binary" {
		t.Fatalf("unexpected content type: %s", downloaded.ContentType)
	}
	if len(downloaded.Data) != 4 {
		t.Fatalf("unexpected data length: %d", len(downloaded.Data))
	}
}

func TestHTTP500TriggersRetriesThenRequestError(t *testing.T) {
	var attempts int32
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(500)
		_, _ = w.Write([]byte("internal boom"))
	})

	_, err := client.GetBalance(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	var reqErr *RequestError
	if !asRequestError(err, &reqErr) {
		t.Fatalf("expected *RequestError, got %T: %v", err, err)
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Fatalf("expected 2 attempts (1 retry), got %d", got)
	}
}

// Small `errors.As` wrappers kept local to the test file to avoid importing
// the stdlib "errors" package purely for type assertions in a handful of
// spots above.
func asTaskError(err error, target **TaskError) bool {
	if e, ok := err.(*TaskError); ok {
		*target = e
		return true
	}
	return false
}

func asTimeoutError(err error, target **TimeoutError) bool {
	if e, ok := err.(*TimeoutError); ok {
		*target = e
		return true
	}
	return false
}

func asAPIError(err error, target **APIError) bool {
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	return false
}

func asRequestError(err error, target **RequestError) bool {
	if e, ok := err.(*RequestError); ok {
		*target = e
		return true
	}
	return false
}
