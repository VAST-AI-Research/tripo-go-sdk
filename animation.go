package tripo3d

import (
	"context"
	"errors"
	"strings"
)

// RigCheckParams are the parameters for POST /v3/animations/rig-check.
type RigCheckParams struct {
	Input string `json:"input"`

	Extra map[string]interface{} `json:"-"`
}

// RigCheck calls POST /v3/animations/rig-check, which determines whether a
// generated model can be rigged and (if so) the recommended skeleton type.
// It returns the resulting task_id.
func (c *Client) RigCheck(ctx context.Context, params RigCheckParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: RigCheck: Input (source task_id) is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/animations/rig-check", body)
}

// RigModelParams are the parameters for POST /v3/animations/rig.
type RigModelParams struct {
	Input     string  `json:"input"`
	RigType   *string `json:"rig_type,omitempty"`
	Spec      *string `json:"spec,omitempty"`
	OutFormat *string `json:"out_format,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// RigModel calls POST /v3/animations/rig (attach a skeleton to a model) and
// returns the resulting task_id.
func (c *Client) RigModel(ctx context.Context, params RigModelParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: RigModel: Input (source task_id) is required")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/animations/rig", body)
}

// RetargetAnimationParams are the parameters for
// POST /v3/animations/retarget. Provide either Animation (single preset) or
// Animations (up to 5).
type RetargetAnimationParams struct {
	Input              string   `json:"input"`
	Animation          *string  `json:"animation,omitempty"`
	Animations         []string `json:"animations,omitempty"`
	OutFormat          *string  `json:"out_format,omitempty"`
	BakeAnimation      *bool    `json:"bake_animation,omitempty"`
	ExportWithGeometry *bool    `json:"export_with_geometry,omitempty"`
	AnimateInPlace     *bool    `json:"animate_in_place,omitempty"`

	Extra map[string]interface{} `json:"-"`
}

// RetargetAnimation calls POST /v3/animations/retarget (apply preset
// animations to a rigged model) and returns the resulting task_id.
func (c *Client) RetargetAnimation(ctx context.Context, params RetargetAnimationParams) (string, error) {
	if strings.TrimSpace(params.Input) == "" {
		return "", errors.New("tripo3d: RetargetAnimation: Input (rigged task_id) is required")
	}
	if params.Animation == nil && len(params.Animations) == 0 {
		return "", errors.New("tripo3d: RetargetAnimation: provide Animation or a non-empty Animations slice")
	}
	if len(params.Animations) > 5 {
		return "", errors.New("tripo3d: RetargetAnimation: at most 5 animations per call")
	}
	body, err := buildPayload(params, params.Extra)
	if err != nil {
		return "", err
	}
	return c.createTask(ctx, "/animations/retarget", body)
}
