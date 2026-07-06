package tripo3d

import (
	"context"
	"errors"
	"strings"
)

// TextToModelParams are the parameters for POST /v3/generation/text-to-model.
type TextToModelParams struct {
	Prompt          string  `json:"prompt"`
	Model           *string `json:"model,omitempty"`
	NegativePrompt  *string `json:"negative_prompt,omitempty"`
	ImageSeed       *int64  `json:"image_seed,omitempty"`
	ModelSeed       *int64  `json:"model_seed,omitempty"`
	TextureSeed     *int64  `json:"texture_seed,omitempty"`
	Texture         *bool   `json:"texture,omitempty"`
	PBR             *bool   `json:"pbr,omitempty"`
	TextureQuality  *string `json:"texture_quality,omitempty"`
	GeometryQuality *string `json:"geometry_quality,omitempty"`
	FaceLimit       *int64  `json:"face_limit,omitempty"`
	AutoSize        *bool   `json:"auto_size,omitempty"`
	Quad            *bool   `json:"quad,omitempty"`
	SmartLowPoly    *bool   `json:"smart_low_poly,omitempty"`
	GenerateParts   *bool   `json:"generate_parts,omitempty"`
	Compress        *string `json:"compress,omitempty"`
	ExportUV        *bool   `json:"export_uv,omitempty"`
	Style           *string `json:"style,omitempty"`

	// Extra carries forward-compatible fields not yet modeled above; they
	// are merged into the JSON payload alongside the typed fields.
	Extra map[string]interface{} `json:"-"`
}

// TextToModel calls POST /v3/generation/text-to-model and returns the
// resulting task_id.
func (c *Client) TextToModel(ctx context.Context, params TextToModelParams) (string, error) {
	if strings.TrimSpace(params.Prompt) == "" {
		return "", errors.New("tripo3d: TextToModel: Prompt is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/text-to-model", body)
}

// ImageToModelParams are the parameters for POST /v3/generation/image-to-model.
type ImageToModelParams struct {
	File               FileDescriptor `json:"file"`
	Model              *string        `json:"model,omitempty"`
	EnableImageAutofix *bool          `json:"enable_image_autofix,omitempty"`
	ModelSeed          *int64         `json:"model_seed,omitempty"`
	TextureSeed        *int64         `json:"texture_seed,omitempty"`
	Texture            *bool          `json:"texture,omitempty"`
	PBR                *bool          `json:"pbr,omitempty"`
	TextureQuality     *string        `json:"texture_quality,omitempty"`
	TextureAlignment   *string        `json:"texture_alignment,omitempty"`
	GeometryQuality    *string        `json:"geometry_quality,omitempty"`
	FaceLimit          *int64         `json:"face_limit,omitempty"`
	AutoSize           *bool          `json:"auto_size,omitempty"`
	Orientation        *string        `json:"orientation,omitempty"`
	Quad               *bool          `json:"quad,omitempty"`
	SmartLowPoly       *bool          `json:"smart_low_poly,omitempty"`
	GenerateParts      *bool          `json:"generate_parts,omitempty"`
	Compress           *string        `json:"compress,omitempty"`
	ExportUV           *bool          `json:"export_uv,omitempty"`
	Style              *string        `json:"style,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// ImageToModel calls POST /v3/generation/image-to-model and returns the
// resulting task_id.
func (c *Client) ImageToModel(ctx context.Context, params ImageToModelParams) (string, error) {
	if params.File.IsEmpty() {
		return "", errors.New("tripo3d: ImageToModel: File is required (url, file_token, or object)")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/image-to-model", body)
}

// MultiviewToModelParams are the parameters for
// POST /v3/generation/multiview-to-model.
//
// Files, when set, must contain exactly 4 items in
// [front, left, back, right] order. Individual items may be a zero-value
// FileDescriptor (empty) except the front view. Mutually exclusive with
// OriginalTaskID.
type MultiviewToModelParams struct {
	Files            *[4]FileDescriptor `json:"files,omitempty"`
	OriginalTaskID   *string            `json:"original_task_id,omitempty"`
	Model            *string            `json:"model,omitempty"`
	ModelSeed        *int64             `json:"model_seed,omitempty"`
	TextureSeed      *int64             `json:"texture_seed,omitempty"`
	Texture          *bool              `json:"texture,omitempty"`
	PBR              *bool              `json:"pbr,omitempty"`
	TextureQuality   *string            `json:"texture_quality,omitempty"`
	TextureAlignment *string            `json:"texture_alignment,omitempty"`
	FaceLimit        *int64             `json:"face_limit,omitempty"`
	AutoSize         *bool              `json:"auto_size,omitempty"`
	Orientation      *string            `json:"orientation,omitempty"`
	Quad             *bool              `json:"quad,omitempty"`
	SmartLowPoly     *bool              `json:"smart_low_poly,omitempty"`
	GenerateParts    *bool              `json:"generate_parts,omitempty"`
	Compress         *string            `json:"compress,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// MultiviewToModel calls POST /v3/generation/multiview-to-model and returns
// the resulting task_id.
func (c *Client) MultiviewToModel(ctx context.Context, params MultiviewToModelParams) (string, error) {
	if params.Files == nil && params.OriginalTaskID == nil {
		return "", errors.New("tripo3d: MultiviewToModel: provide Files ([front, left, back, right]) or OriginalTaskID")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/multiview-to-model", body)
}

// TextToImageParams are the parameters for POST /v3/generation/text-to-image.
type TextToImageParams struct {
	Prompt string  `json:"prompt"`
	Model  *string `json:"model,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// TextToImage calls POST /v3/generation/text-to-image and returns the
// resulting task_id.
func (c *Client) TextToImage(ctx context.Context, params TextToImageParams) (string, error) {
	if strings.TrimSpace(params.Prompt) == "" {
		return "", errors.New("tripo3d: TextToImage: Prompt is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/text-to-image", body)
}

// ImageToImageParams are the parameters for POST /v3/generation/image-to-image.
type ImageToImageParams struct {
	File   *FileDescriptor `json:"file,omitempty"`
	Prompt *string         `json:"prompt,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// ImageToImage calls POST /v3/generation/image-to-image (image style / edit
// transformation) and returns the resulting task_id.
func (c *Client) ImageToImage(ctx context.Context, params ImageToImageParams) (string, error) {
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/image-to-image", body)
}

// ImageToMultiviewParams are the parameters for
// POST /v3/generation/image-to-multiview.
type ImageToMultiviewParams struct {
	File *FileDescriptor `json:"file,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// ImageToMultiview calls POST /v3/generation/image-to-multiview and returns
// the resulting task_id.
func (c *Client) ImageToMultiview(ctx context.Context, params ImageToMultiviewParams) (string, error) {
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/image-to-multiview", body)
}

// EditMultiviewParams are the parameters for
// POST /v3/generation/edit-multiview.
type EditMultiviewParams struct {
	OriginalTaskID *string `json:"original_task_id,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// EditMultiview calls POST /v3/generation/edit-multiview (refine a
// previously generated multiview set) and returns the resulting task_id.
func (c *Client) EditMultiview(ctx context.Context, params EditMultiviewParams) (string, error) {
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/generation/edit-multiview", body)
}
