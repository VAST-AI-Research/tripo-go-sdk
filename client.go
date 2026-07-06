// Package tripo3d is an unofficial Go SDK for the Tripo3D v3 API
// (https://developers.tripo3d.com/en/docs/introduction) — an AI 3D
// generation platform covering text-to-3D, image-to-3D, multiview-to-3D,
// re-texturing, mesh editing, auto-rigging, and animation retargeting.
//
//	client, err := tripo3d.NewClient(tripo3d.ClientOptions{}) // reads TRIPO_API_KEY
//	taskID, err := client.TextToModel(ctx, tripo3d.TextToModelParams{Prompt: "a cute cat"})
//	task, err := client.WaitForTask(ctx, taskID, tripo3d.WaitOptions{})
//	fmt.Println(task.PrimaryModelURL())
package tripo3d

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"
)

// ClientOptions configures a Client. All fields are optional; APIKey falls
// back to the TRIPO_API_KEY environment variable when unset.
type ClientOptions struct {
	// APIKey authenticates requests. Falls back to the TRIPO_API_KEY
	// environment variable when empty.
	APIKey string
	// BaseURL overrides the API endpoint. Defaults to DefaultBaseURL.
	BaseURL string
	// HTTPClient overrides the underlying *http.Client. Defaults to
	// http.DefaultClient's zero value (a fresh client with no timeout of
	// its own — per-request timeouts are enforced via Timeout instead).
	HTTPClient *http.Client
	// Timeout bounds each individual HTTP request (not the whole
	// WaitForTask polling loop). Defaults to 60s.
	Timeout time.Duration
	// Retries is the number of extra attempts made on transient network
	// errors or 5xx/429 responses. Defaults to 2.
	//
	// Quirk: because Go's zero value for int is 0, and 0 is also a
	// perfectly valid "use the default" sentinel here, pass -1 to
	// explicitly disable retries.
	Retries int
	// UserAgent overrides the default "tripo3d-sdk-go/<version>" header.
	UserAgent string
}

// Client is the entry point for the Tripo3D v3 API.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
	retries    int
	userAgent  string
}

// SDKVersion is the current tripo3d-sdk-go release version, also used as
// part of the default User-Agent header.
const SDKVersion = "0.1.0"

// NewClient builds a Client, reading TRIPO_API_KEY from the environment
// when opts.APIKey is empty.
func NewClient(opts ClientOptions) (*Client, error) {
	apiKey := opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("TRIPO_API_KEY")
	}
	if apiKey == "" {
		return nil, errors.New("tripo3d: an API key is required; set ClientOptions.APIKey or the TRIPO_API_KEY environment variable")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	retries := opts.Retries
	switch {
	case retries == 0:
		retries = 2
	case retries < 0:
		retries = 0
	}

	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = "tripo3d-sdk-go/" + SDKVersion
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: httpClient,
		timeout:    timeout,
		retries:    retries,
		userAgent:  userAgent,
	}, nil
}

// ─────────────────────────────── Account ───────────────────────────────────

// GetBalance calls GET /v3/account/balance.
func (c *Client) GetBalance(ctx context.Context) (*Balance, error) {
	var balance Balance
	if err := c.doJSON(ctx, http.MethodGet, "/account/balance", nil, &balance); err != nil {
		return nil, err
	}
	return &balance, nil
}

// ─────────────────────────────── Files ─────────────────────────────────────

// UploadFile calls POST /v3/files, uploading a raw file and returning its
// file_token.
func (c *Client) UploadFile(ctx context.Context, data []byte, filename, contentType string) (*UploadedFile, error) {
	if filename == "" {
		filename = "upload.bin"
	}
	var uploaded UploadedFile
	if err := c.doMultipart(ctx, "/files", "file", filename, contentType, data, &uploaded); err != nil {
		return nil, err
	}
	return &uploaded, nil
}

// ────────────────────────── Task management ────────────────────────────────

// GetTask calls GET /v3/tasks/{task_id}.
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	if taskID == "" {
		return nil, errors.New("tripo3d: GetTask: taskID is required")
	}
	var task Task
	if err := c.doJSON(ctx, http.MethodGet, "/tasks/"+pathEscape(taskID), nil, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// ListTasks calls POST /v3/tasks/list to batch-query multiple tasks in one
// round trip.
func (c *Client) ListTasks(ctx context.Context, taskIDs []string) ([]Task, error) {
	if len(taskIDs) == 0 {
		return nil, errors.New("tripo3d: ListTasks: taskIDs must be non-empty")
	}
	var result struct {
		Tasks []Task `json:"tasks"`
	}
	body := map[string]interface{}{"task_ids": taskIDs}
	if err := c.doJSON(ctx, http.MethodPost, "/tasks/list", body, &result); err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

// WaitOptions configures Client.WaitForTask.
type WaitOptions struct {
	// PollInterval is the delay between polling attempts. Defaults to 2s.
	PollInterval time.Duration
	// Timeout bounds the overall polling loop. Zero means wait
	// indefinitely (until ctx is cancelled).
	Timeout time.Duration
	// IgnoreFailure, when true, returns the task even if its terminal
	// status isn't "success", instead of returning a *TaskError.
	IgnoreFailure bool
	// OnProgress, if set, is invoked after every poll (including the
	// final one) with the latest task snapshot.
	OnProgress func(*Task)
}

// WaitForTask polls GET /v3/tasks/{task_id} until the task reaches a
// terminal state, or the context is cancelled / opts.Timeout elapses.
//
// By default, a non-"success" terminal status is surfaced as a *TaskError;
// set opts.IgnoreFailure to receive the task without an error instead.
func (c *Client) WaitForTask(ctx context.Context, taskID string, opts WaitOptions) (*Task, error) {
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	started := time.Now()

	for {
		task, err := c.GetTask(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if opts.OnProgress != nil {
			opts.OnProgress(task)
		}

		if task.Status.IsTerminal() {
			if !opts.IgnoreFailure && !task.Status.IsSuccess() {
				return task, &TaskError{Task: task}
			}
			return task, nil
		}

		if opts.Timeout > 0 && time.Since(started) >= opts.Timeout {
			return nil, &TimeoutError{TaskID: taskID, Timeout: opts.Timeout}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// ────────────────────────────── Downloads ───────────────────────────────────

// DownloadModel fetches the primary model URL of a completed task.
// Returns (nil, nil) when the task has no model output.
func (c *Client) DownloadModel(ctx context.Context, task *Task) (*DownloadedModel, error) {
	u := task.PrimaryModelURL()
	if u == "" {
		return nil, nil
	}
	data, contentType, err := c.getRaw(ctx, u)
	if err != nil {
		return nil, err
	}
	return &DownloadedModel{URL: u, ContentType: contentType, Data: data}, nil
}
