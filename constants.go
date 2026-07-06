package tripo3d

// DefaultBaseURL is the default REST endpoint for the Tripo3D v3 openapi
// service.
const DefaultBaseURL = "https://openapi.tripo3d.com/v3"

// TaskStatus is the lifecycle status of a task, as returned by
// GET /v3/tasks/{task_id}.
type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSuccess   TaskStatus = "success"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusUnknown   TaskStatus = "unknown"
	TaskStatusBanned    TaskStatus = "banned"
	TaskStatusExpired   TaskStatus = "expired"
)

// IsTerminal reports whether a task in this status will not change further.
func (s TaskStatus) IsTerminal() bool {
	switch s {
	case TaskStatusSuccess, TaskStatusFailed, TaskStatusCancelled, TaskStatusBanned, TaskStatusExpired:
		return true
	default:
		return false
	}
}

// IsSuccess reports whether this status represents a successful completion.
func (s TaskStatus) IsSuccess() bool {
	return s == TaskStatusSuccess
}

// Animation is a preset animation identifier accepted by
// POST /v3/animations/retarget. Combine at most 5 in a single call.
type Animation string

const (
	AnimationIdle            Animation = "preset:idle"
	AnimationWalk            Animation = "preset:walk"
	AnimationRun             Animation = "preset:run"
	AnimationDive            Animation = "preset:dive"
	AnimationClimb           Animation = "preset:climb"
	AnimationJump            Animation = "preset:jump"
	AnimationSlash           Animation = "preset:slash"
	AnimationShoot           Animation = "preset:shoot"
	AnimationHurt            Animation = "preset:hurt"
	AnimationFall            Animation = "preset:fall"
	AnimationTurn            Animation = "preset:turn"
	AnimationQuadrupedWalk   Animation = "preset:quadruped:walk"
	AnimationHexapodWalk     Animation = "preset:hexapod:walk"
	AnimationOctopodWalk     Animation = "preset:octopod:walk"
	AnimationSerpentineMarch Animation = "preset:serpentine:march"
	AnimationAquaticMarch    Animation = "preset:aquatic:march"
)

// RigType is a skeleton topology for POST /v3/animations/rig.
type RigType string

const (
	RigTypeBiped      RigType = "biped"
	RigTypeQuadruped  RigType = "quadruped"
	RigTypeHexapod    RigType = "hexapod"
	RigTypeOctopod    RigType = "octopod"
	RigTypeAvian      RigType = "avian"
	RigTypeSerpentine RigType = "serpentine"
	RigTypeAquatic    RigType = "aquatic"
	RigTypeOthers     RigType = "others"
)

// RigSpec controls bone naming/hierarchy conventions.
type RigSpec string

const (
	RigSpecMixamo RigSpec = "mixamo"
	RigSpecTripo  RigSpec = "tripo"
)

// AnimOutFormat is the output format for rig / retarget tasks.
type AnimOutFormat string

const (
	AnimOutFormatGLB AnimOutFormat = "glb"
	AnimOutFormatFBX AnimOutFormat = "fbx"
)

// OutputFormat is a model conversion output format accepted by
// POST /v3/models/convert.
type OutputFormat string

const (
	OutputFormatGLTF    OutputFormat = "GLTF"
	OutputFormatGLB     OutputFormat = "GLB"
	OutputFormatUSDZ    OutputFormat = "USDZ"
	OutputFormatFBX     OutputFormat = "FBX"
	OutputFormatOBJ     OutputFormat = "OBJ"
	OutputFormatSTL     OutputFormat = "STL"
	OutputFormatThreeMF OutputFormat = "3MF"
)

// TextureFormat is a texture image format supported by the conversion API.
type TextureFormat string

const (
	TextureFormatBMP     TextureFormat = "BMP"
	TextureFormatDPX     TextureFormat = "DPX"
	TextureFormatHDR     TextureFormat = "HDR"
	TextureFormatJPEG    TextureFormat = "JPEG"
	TextureFormatOpenEXR TextureFormat = "OPEN_EXR"
	TextureFormatPNG     TextureFormat = "PNG"
	TextureFormatTarga   TextureFormat = "TARGA"
	TextureFormatTIFF    TextureFormat = "TIFF"
	TextureFormatWebP    TextureFormat = "WEBP"
)

// ModelVersion lists the `model` values (product lines) exposed by the v3
// API. Kept as plain strings (not a closed type) since Tripo3D regularly
// ships new model versions.
const (
	ModelVersionH31     = "v3.1-20260211"
	ModelVersionH30     = "v3.0-20250812"
	ModelVersionH25     = "v2.5-20250123"
	ModelVersionH20     = "v2.0-20240919"
	ModelVersionP1      = "P1-20260311"
	ModelVersionTurboV1 = "Turbo-v1.0-20250506"
)
