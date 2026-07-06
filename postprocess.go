package tripo3d

import (
	"context"
	"errors"
	"strings"
)

// TextureModelParams are the parameters for POST /v3/models/texture.
type TextureModelParams struct {
	Input            string          `json:"input"`
	Texture          *bool           `json:"texture,omitempty"`
	PBR              *bool           `json:"pbr,omitempty"`
	ModelSeed        *int64          `json:"model_seed,omitempty"`
	TextureSeed      *int64          `json:"texture_seed,omitempty"`
	TextureQuality   *string         `json:"texture_quality,omitempty"`
	TextureAlignment *string         `json:"texture_alignment,omitempty"`
	TextPrompt       *string         `json:"text_prompt,omitempty"`
	ImagePrompt      *FileDescriptor `json:"image_prompt,omitempty"`
	StyleImage       *FileDescriptor `json:"style_image,omitempty"`
	Compress         *string         `json:"compress,omitempty"`
	Bake             *bool           `json:"bake,omitempty"`
	PartNames        []string        `json:"part_names,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// TextureModel calls POST /v3/models/texture (re-texture an existing model)
// and returns the resulting task_id.
func (c *Client) TextureModel(ctx context.Context, params TextureModelParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: TextureModel: Input (source task_id) is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/models/texture", body)
}

// ConvertModelParams are the parameters for POST /v3/models/convert.
type ConvertModelParams struct {
	Input                  string   `json:"input"`
	Format                 string   `json:"format"`
	Quad                   *bool    `json:"quad,omitempty"`
	FaceLimit              *int64   `json:"face_limit,omitempty"`
	TextureSize            *int64   `json:"texture_size,omitempty"`
	TextureFormat          *string  `json:"texture_format,omitempty"`
	FlattenBottom          *bool    `json:"flatten_bottom,omitempty"`
	FlattenBottomThreshold *float64 `json:"flatten_bottom_threshold,omitempty"`
	PivotToCenterBottom    *bool    `json:"pivot_to_center_bottom,omitempty"`
	WithAnimation          *bool    `json:"with_animation,omitempty"`
	PackUV                 *bool    `json:"pack_uv,omitempty"`
	ForceSymmetry          *bool    `json:"force_symmetry,omitempty"`
	Bake                   *bool    `json:"bake,omitempty"`
	PartNames              []string `json:"part_names,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// ConvertModel calls POST /v3/models/convert (convert a completed model to
// another format) and returns the resulting task_id.
func (c *Client) ConvertModel(ctx context.Context, params ConvertModelParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: ConvertModel: Input is required")
	}
	if strings.TrimSpace(params.Format) == "" {
		return "", errors.New("tripo3d: ConvertModel: Format is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/models/convert", body)
}

// SegmentMeshParams are the parameters for POST /v3/mesh/segment.
type SegmentMeshParams struct {
	Input string  `json:"input"`
	Model *string `json:"model,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// SegmentMesh calls POST /v3/mesh/segment (semantic segmentation of a mesh)
// and returns the resulting task_id.
func (c *Client) SegmentMesh(ctx context.Context, params SegmentMeshParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: SegmentMesh: Input is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/mesh/segment", body)
}

// CompleteMeshParams are the parameters for POST /v3/mesh/complete.
type CompleteMeshParams struct {
	Input     string   `json:"input"`
	Model     *string  `json:"model,omitempty"`
	PartNames []string `json:"part_names,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// CompleteMesh calls POST /v3/mesh/complete (mesh completion / repair) and
// returns the resulting task_id.
func (c *Client) CompleteMesh(ctx context.Context, params CompleteMeshParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: CompleteMesh: Input is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/mesh/complete", body)
}

// DecimateMeshParams are the parameters for POST /v3/mesh/decimate.
type DecimateMeshParams struct {
	Input     string   `json:"input"`
	Model     *string  `json:"model,omitempty"`
	FaceLimit *int64   `json:"face_limit,omitempty"`
	Quad      *bool    `json:"quad,omitempty"`
	Bake      *bool    `json:"bake,omitempty"`
	PartNames []string `json:"part_names,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// DecimateMesh calls POST /v3/mesh/decimate (retopology / face-count
// reduction) and returns the resulting task_id.
func (c *Client) DecimateMesh(ctx context.Context, params DecimateMeshParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: DecimateMesh: Input is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/mesh/decimate", body)
}
