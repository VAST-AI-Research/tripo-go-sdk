package tripo3d

import (
	"encoding/json"
	"strings"
)

// envelope is the standard `{ code, data, message, suggestion }` response
// wrapper used by every Tripo3D v3 endpoint.
type envelope struct {
	Code       int             `json:"code"`
	Data       json.RawMessage `json:"data,omitempty"`
	Message    string          `json:"message,omitempty"`
	Suggestion string          `json:"suggestion,omitempty"`
}

// ObjectRef is a bucket/key pair for pre-uploaded assets (STS-style upload).
type ObjectRef struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// FileDescriptor is the shape accepted by every endpoint that takes an
// image or model file as input. Exactly one of Ref, FileToken, URL, or
// Object should normally be set.
type FileDescriptor struct {
	// Ref is a bare reference whose kind the server infers: a public URL, a
	// file_token from Client.UploadFile, or the task_id of an earlier
	// generation task whose output should be reused. It marshals as a plain
	// JSON string, the form the v3 API documents.
	Ref       string     `json:"-"`
	FileToken string     `json:"file_token,omitempty"`
	URL       string     `json:"url,omitempty"`
	Object    *ObjectRef `json:"object,omitempty"`
	Type      string     `json:"type,omitempty"`
}

// IsEmpty reports whether none of the descriptor's fields are populated.
func (f FileDescriptor) IsEmpty() bool {
	return f.Ref == "" && f.FileToken == "" && f.URL == "" && f.Object == nil
}

// MarshalJSON emits a bare string for a Ref-only descriptor (and for an
// empty one, which the multiview endpoints use to skip a view), and the
// explicit object form otherwise.
func (f FileDescriptor) MarshalJSON() ([]byte, error) {
	if f.Ref != "" {
		return json.Marshal(f.Ref)
	}
	if f.IsEmpty() {
		return []byte(`""`), nil
	}
	type alias FileDescriptor
	return json.Marshal(alias(f))
}

// UnmarshalJSON accepts either the bare string form or the object form.
func (f *FileDescriptor) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FileDescriptor{Ref: s}
		return nil
	}
	type alias FileDescriptor
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*f = FileDescriptor(a)
	return nil
}

// File builds a FileDescriptor from a bare reference — a URL, a
// file_token, or a task_id — and lets the server infer which it is.
func File(ref string) FileDescriptor {
	return FileDescriptor{Ref: ref}
}

// FileURL builds a FileDescriptor that explicitly references a publicly
// accessible image or model URL.
func FileURL(url string) FileDescriptor {
	return FileDescriptor{URL: url}
}

// FileToken builds a FileDescriptor that explicitly references a token
// returned by Client.UploadFile.
func FileToken(token string) FileDescriptor {
	return FileDescriptor{FileToken: token}
}

// MultiviewPrompt is a single per-view edit instruction for
// Client.EditMultiview.
type MultiviewPrompt struct {
	// Prompt describes the desired change, e.g. "change the shirt color to
	// red".
	Prompt string `json:"prompt"`
	// View is the angle to apply the edit to: ViewFront, ViewLeft,
	// ViewBack, or ViewRight.
	View string `json:"view"`
}

// Task is a task snapshot as returned by GET /v3/tasks/{task_id} and
// POST /v3/tasks/list.
type Task struct {
	TaskID   string          `json:"task_id"`
	Type     string          `json:"type"`
	Status   TaskStatus      `json:"status"`
	Progress int             `json:"progress,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
	Output   *TaskOutput     `json:"output,omitempty"`

	// CreditsConsumed is a decimal with up to two places (e.g. 48.00), so
	// it is deliberately a float rather than an integer — VIP discounts
	// produce fractional values that integer parsing would truncate.
	CreditsConsumed float64 `json:"credits_consumed,omitempty"`

	// CreatedAt and CompletedAt are ISO 8601 timestamps. CompletedAt is
	// empty until the task reaches a terminal status.
	CreatedAt   string `json:"created_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`

	RunningLeftTime int64 `json:"running_left_time,omitempty"`
	QueuingNum      int64 `json:"queuing_num,omitempty"`

	// ErrorCode and ErrorMessage are populated when Status is
	// TaskStatusFailed.
	ErrorCode    int64  `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Raw holds the complete, unparsed JSON object for this task, so callers
	// can reach fields this struct doesn't model yet.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the known fields while also retaining the full raw
// payload in Raw, so newly added API fields remain accessible.
func (t *Task) UnmarshalJSON(data []byte) error {
	type alias Task
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*t = Task(a)
	t.Raw = append(json.RawMessage{}, data...)
	return nil
}

// PrimaryModelURL returns the best-effort "main" model URL for this task,
// checking the common output fields in priority order. Returns "" if the
// task has no model output.
func (t *Task) PrimaryModelURL() string {
	if t == nil {
		return ""
	}
	return t.Output.PrimaryModelURL()
}

// TaskOutput is the `output` object of a completed task. Fields vary by
// task type, so everything is optional.
type TaskOutput struct {
	Model            string   `json:"model,omitempty"`
	ModelURL         string   `json:"model_url,omitempty"`
	ModelURLs        []string `json:"model_urls,omitempty"`
	BaseModel        string   `json:"base_model,omitempty"`
	PBRModel         string   `json:"pbr_model,omitempty"`
	RenderedImage    string   `json:"rendered_image,omitempty"`
	RenderedImageURL string   `json:"rendered_image_url,omitempty"`

	// GeneratedImageURL is the output of the text-to-image and
	// image-to-image endpoints. The 3D generation endpoints also populate
	// it with the reference image they synthesised internally.
	GeneratedImageURL string `json:"generated_image_url,omitempty"`

	Riggable *bool  `json:"riggable,omitempty"`
	RigType  string `json:"rig_type,omitempty"`

	// Raw holds the complete, unparsed JSON object for this output.
	Raw json.RawMessage `json:"-"`
}

// UnmarshalJSON decodes the known fields while also retaining the full raw
// payload in Raw.
func (o *TaskOutput) UnmarshalJSON(data []byte) error {
	type alias TaskOutput
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*o = TaskOutput(a)
	o.Raw = append(json.RawMessage{}, data...)
	return nil
}

// PrimaryModelURL returns the best-effort "main" model URL, checking the
// common output fields in priority order. Safe to call on a nil receiver.
func (o *TaskOutput) PrimaryModelURL() string {
	if o == nil {
		return ""
	}
	if o.ModelURL != "" {
		return o.ModelURL
	}
	if o.Model != "" {
		return o.Model
	}
	if o.PBRModel != "" {
		return o.PBRModel
	}
	if o.BaseModel != "" {
		return o.BaseModel
	}
	if len(o.ModelURLs) > 0 {
		return o.ModelURLs[0]
	}
	return ""
}

// IsRiggable reports the rig-check verdict, defaulting to false when unset.
func (o *TaskOutput) IsRiggable() bool {
	return o != nil && o.Riggable != nil && *o.Riggable
}

// Balance is the GET /v3/account/balance response payload.
type Balance struct {
	Balance float64 `json:"balance"`
	Frozen  float64 `json:"frozen,omitempty"`
}

// UploadedFile is the POST /v3/files response payload.
type UploadedFile struct {
	FileToken string `json:"file_token"`
}

// DownloadedModel is the result of Client.DownloadModel.
type DownloadedModel struct {
	URL         string
	ContentType string
	Data        []byte
}

// Extension returns the lower-case file extension of the downloaded model,
// without the leading dot (e.g. "glb", "fbx"), or "" if the URL carries
// none.
//
// Do not assume GLB: setting Quad on a generation task forces FBX output,
// and Client.ConvertModel emits whichever format was requested.
func (d *DownloadedModel) Extension() string {
	if d == nil {
		return ""
	}
	path := d.URL
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if i := strings.LastIndex(path, "/"); i >= 0 {
		path = path[i+1:]
	}
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return ""
	}
	return strings.ToLower(path[i+1:])
}

// Filename returns a download-ready "<name>.<ext>" for this model, falling
// back to "glb" when the URL carries no extension.
func (d *DownloadedModel) Filename(name string) string {
	ext := d.Extension()
	if ext == "" {
		ext = "glb"
	}
	return name + "." + ext
}
