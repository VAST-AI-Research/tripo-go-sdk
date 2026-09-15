package tripo3d

// DefaultBaseURL is the China mainland REST endpoint for the Tripo3D v3
// openapi service. For overseas / global traffic use
// "https://openapi.tripo3d.ai/v3" via ClientOptions.BaseURL.
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

// ModelVersion lists the `model` values (product lines) accepted by the 3D
// generation endpoints. Kept as plain strings (not a closed type) since
// Tripo3D regularly ships new model versions.
const (
	ModelVersionH31 = "v3.1-20260211"
	ModelVersionH30 = "v3.0-20250812"
	ModelVersionH25 = "v2.5-20250123"
	ModelVersionP1  = "P1-20260311"
	// ModelVersionP2 is the next-generation P series. It is the only model
	// that accepts Quad, and unlike P1 it supports face limits up to
	// 50,000 triangles (25,000 with Quad). Preview.
	ModelVersionP2 = "P2-20260801"
)

// ImageModel lists the `model` values accepted by
// POST /v3/generation/text-to-image and POST /v3/generation/image-to-image.
const (
	ImageModelSeedreamV5          = "seedream_v5"
	ImageModelBanana              = "banana"
	ImageModelBananaPro           = "banana_pro"
	ImageModelBanana2             = "banana2"
	ImageModelChatImage2          = "chat_image_2"
	ImageModelChatImage25Flare    = "chat_image_2.5_flare"
	ImageModelChatImage25Sunburst = "chat_image_2.5_sunburst"
)

// ImageQuality is the `quality` render tier for image generation. Only
// ImageModelChatImage2 and the 2.5 models accept it; any other model
// rejects the request outright. Omitting it is equivalent to
// ImageQualityLow — note this differs from OpenAI's own `auto` default,
// which is not supported here.
//
// ImageQualityXHigh and ImageQualityMax are exclusive to the 2.5 models.
// High and above cost extra credits and take noticeably longer.
const (
	ImageQualityLow    = "low"
	ImageQualityMedium = "medium"
	ImageQualityHigh   = "high"
	ImageQualityXHigh  = "xhigh"
	ImageQualityMax    = "max"
)

// ImageBackground is the `background` handling mode, supported only by the
// 2.5 models. Other models ignore it rather than rejecting the request.
// ImageBackgroundTransparent requires ImageFormatPNG.
const (
	ImageBackgroundAuto        = "auto"
	ImageBackgroundOpaque      = "opaque"
	ImageBackgroundTransparent = "transparent"
)

// ImageFormat is the `output_format` of a generated image.
const (
	ImageFormatPNG  = "png"
	ImageFormatJPEG = "jpeg"
)

// AspectRatio is the `aspect_ratio` of a generated image. Only the banana
// models accept it; seedream and chat_image size their output via the
// `size` parameter instead.
//
// The 1:8, 1:4, 4:1 and 8:1 ratios are exclusive to ImageModelBanana2.
const (
	AspectRatio1x1  = "1:1"
	AspectRatio2x3  = "2:3"
	AspectRatio3x2  = "3:2"
	AspectRatio3x4  = "3:4"
	AspectRatio4x3  = "4:3"
	AspectRatio4x5  = "4:5"
	AspectRatio5x4  = "5:4"
	AspectRatio9x16 = "9:16"
	AspectRatio16x9 = "16:9"
	AspectRatio21x9 = "21:9"
	AspectRatio1x8  = "1:8"
	AspectRatio1x4  = "1:4"
	AspectRatio4x1  = "4:1"
	AspectRatio8x1  = "8:1"
)

// ImageTemplate is the `template` preset for image generation. Setting one
// makes `prompt` optional.
//
// TemplateAssetExtraction is accepted only by text-to-image, and
// Template3DEnhance only by image-to-image; the rest work on both.
const (
	TemplateAssetExtraction     = "asset_extraction"
	TemplateCharacterCompletion = "character_completion"
	TemplateTPose               = "t_pose"
	TemplateVariants            = "variants"
	TemplateFigure              = "figure"
	Template3DEnhance           = "3d_enhance"
)

// ExportOrientation is the `export_orientation` (forward axis) of a
// generated model.
//
// It applies to that generation only. If the model will be fed into
// post-processing (texture, rig, retarget, convert), leave this unset and
// reorient in the last step via Client.ConvertModel instead — a
// wrongly-oriented post-processing result still reports success rather
// than raising an error.
const (
	ExportOrientationPlusX  = "+x"
	ExportOrientationMinusX = "-x"
	ExportOrientationPlusY  = "+y"
	ExportOrientationMinusY = "-y"
)

// TextureQuality is the `texture_quality` level for 3D generation.
const (
	TextureQualityStandard = "standard"
	TextureQualityDetailed = "detailed"
	TextureQualityExtreme  = "extreme"
)

// GeometryQuality is the `geometry_quality` level for 3D generation. Only
// effective for model versions >= ModelVersionH30; do not send it with
// ModelVersionH25.
const (
	GeometryQualityStandard = "standard"
	GeometryQualityDetailed = "detailed"
)

// TextureAlignment is the `texture_alignment` priority for image- and
// multiview-based 3D generation.
const (
	TextureAlignmentOriginalImage = "original_image"
	TextureAlignmentGeometry      = "geometry"
)

// Orientation is the `orientation` of a generated model relative to the
// input image. Only effective when texturing is enabled.
const (
	OrientationDefault    = "default"
	OrientationAlignImage = "align_image"
)

// View identifies one of the four canonical camera angles used by the
// multiview endpoints.
const (
	ViewFront = "front"
	ViewLeft  = "left"
	ViewBack  = "back"
	ViewRight = "right"
)
