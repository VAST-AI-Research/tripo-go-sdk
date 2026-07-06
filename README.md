# tripo3d-sdk (Go)

**English** · [简体中文](./README.zh-CN.md)

An unofficial **Go SDK** for the [Tripo3D v3 API](https://developers.tripo3d.com/zh/docs/introduction) — a full AI 3D generation platform covering text-to-3D, image-to-3D, multiview-to-3D, re-texturing, mesh editing, auto-rigging and animation retargeting.

- Zero third-party dependencies — built entirely on `net/http` and the standard library.
- `context.Context` on every method for cancellation and deadlines.
- Automatic retries on transient network / 5xx errors, honoring `Retry-After`.
- Typed error values (`*APIError`, `*TaskError`, `*TimeoutError`, `*RequestError`) usable with `errors.As`.
- `WaitForTask` poller with a progress callback.
- Sibling SDK to [`tripo3d-sdk-js`](../tripo3d-sdk-js) and [`tripo3d-sdk-rust`](../tripo3d-sdk-rust) — same API surface, idiomatic per language.

> Base URL: `https://openapi.tripo3d.com/v3` — this SDK targets the **v3** REST endpoints under `developers.tripo3d.com`, not the older `/v2/openapi/task` endpoint.

---

## Installation

This module is not published to a public registry yet. Use a `replace` directive or `go mod edit`:

```bash
go mod edit -replace github.com/your-org/tripo3d-sdk-go=../tripo3d-sdk-go
go mod tidy
```

Or vendor it directly by copying this directory into your module and adjusting the import path.

Create an API key on the [Tripo console](https://platform.tripo3d.ai/) and export it:

```bash
export TRIPO_API_KEY="tsk_..."
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	tripo3d "github.com/your-org/tripo3d-sdk-go"
)

func main() {
	client, err := tripo3d.NewClient(tripo3d.ClientOptions{}) // reads TRIPO_API_KEY
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	taskID, err := client.TextToModel(ctx, tripo3d.TextToModelParams{
		Prompt:         "a cute red panda holding bamboo",
		Model:          tripo3d.String(tripo3d.ModelVersionH31),
		Texture:        tripo3d.Bool(true),
		PBR:            tripo3d.Bool(true),
		TextureQuality: tripo3d.String("detailed"),
	})
	if err != nil {
		log.Fatal(err)
	}

	task, err := client.WaitForTask(ctx, taskID, tripo3d.WaitOptions{
		PollInterval: 2 * time.Second,
		OnProgress: func(t *tripo3d.Task) {
			fmt.Printf("%s — %d%%\n", t.Status, t.Progress)
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Model URL:", task.PrimaryModelURL())
}
```

> ⚠️ Model URLs expire ~5 minutes after task completion — download them right away. See `client.DownloadModel(ctx, task)`.

---

## Client options

```go
tripo3d.NewClient(tripo3d.ClientOptions{
	APIKey:     "",           // defaults to TRIPO_API_KEY env var
	BaseURL:    "",           // default: https://openapi.tripo3d.com/v3
	HTTPClient: nil,          // default: &http.Client{}
	Timeout:    0,            // per-request timeout, default 60s
	Retries:    0,            // extra attempts on 5xx / network errors; 0 = default (2), -1 = disabled
	UserAgent:  "",
})
```

---

## API reference

Every generation method returns a `task_id` (`string`). Use `WaitForTask` to await the terminal result.

### Generation

| Method | Endpoint | Description |
| --- | --- | --- |
| `TextToModel(ctx, params)` | `POST /generation/text-to-model` | Text → 3D model |
| `ImageToModel(ctx, params)` | `POST /generation/image-to-model` | Single image → 3D model |
| `MultiviewToModel(ctx, params)` | `POST /generation/multiview-to-model` | 4 views `[front, left, back, right]` → 3D model |
| `TextToImage(ctx, params)` | `POST /generation/text-to-image` | Concept image from text |
| `ImageToImage(ctx, params)` | `POST /generation/image-to-image` | Image style / edit |
| `ImageToMultiview(ctx, params)` | `POST /generation/image-to-multiview` | Image → 4-view sheet |
| `EditMultiview(ctx, params)` | `POST /generation/edit-multiview` | Refine multiview output |

### Model post-processing

| Method | Endpoint | Description |
| --- | --- | --- |
| `TextureModel(ctx, params)` | `POST /models/texture` | Re-texture an existing model |
| `ConvertModel(ctx, params)` | `POST /models/convert` | Convert to GLTF / FBX / OBJ / STL / USDZ / 3MF |
| `SegmentMesh(ctx, params)` | `POST /mesh/segment` | Semantic segmentation |
| `CompleteMesh(ctx, params)` | `POST /mesh/complete` | Mesh completion / repair |
| `DecimateMesh(ctx, params)` | `POST /mesh/decimate` | Retopology / face-count reduction |

### Animation

| Method | Endpoint | Description |
| --- | --- | --- |
| `RigCheck(ctx, params)` | `POST /animations/rig-check` | Detect whether a model is riggable |
| `RigModel(ctx, params)` | `POST /animations/rig` | Attach a skeleton |
| `RetargetAnimation(ctx, params)` | `POST /animations/retarget` | Apply preset animations |

### Utility

| Method | Endpoint | Description |
| --- | --- | --- |
| `GetTask(ctx, taskID)` | `GET /tasks/{task_id}` | Fetch a task snapshot |
| `ListTasks(ctx, taskIDs)` | `POST /tasks/list` | Batch task query |
| `WaitForTask(ctx, taskID, opts)` | — | Poll until terminal state |
| `UploadFile(ctx, data, filename, contentType)` | `POST /files` | Upload a raw file and get a `file_token` |
| `GetBalance(ctx)` | `GET /account/balance` | Account credit balance |
| `DownloadModel(ctx, task)` | — | Download the primary model URL into a `[]byte` |

---

## Passing images / files

Any field of type `FileDescriptor` (or `*FileDescriptor`) can be built with `tripo3d.File(...)`, which auto-detects URLs vs. bare file tokens:

```go
f1 := tripo3d.File("https://example.com/hero.png")   // -> FileDescriptor{URL: "..."}
f2 := tripo3d.File("8f2a4c...")                       // -> FileDescriptor{FileToken: "..."}
f3 := tripo3d.FileDescriptor{Object: &tripo3d.ObjectRef{Bucket: "tripo-data", Key: "uploads/abc.png"}}
```

Upload a local buffer to get a `file_token`:

```go
data, _ := os.ReadFile("./hero.png")
uploaded, err := client.UploadFile(ctx, data, "hero.png", "image/png")

taskID, err := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	File: tripo3d.File(uploaded.FileToken),
})
```

### Optional fields and pointer helpers

Optional scalar fields are pointers (so the SDK can distinguish "not set" from the zero value). Use the `tripo3d.Bool`, `tripo3d.String`, `tripo3d.Int64`, and `tripo3d.Float64` helpers to fill them:

```go
tripo3d.TextToModelParams{
	Prompt:  "a cat",
	Texture: tripo3d.Bool(true),
	Model:   tripo3d.String(tripo3d.ModelVersionH31),
}
```

Every `*Params` struct also has an `Extra map[string]interface{}` field for forward-compatible passthrough of fields the SDK doesn't model yet.

---

## End-to-end pipeline: game-ready character

```go
// 1. Image -> 3D (low-poly P1 topology, mobile/game friendly)
modelID, _ := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	File:      tripo3d.File("https://example.com/hero.png"),
	Model:     tripo3d.String(tripo3d.ModelVersionP1),
	FaceLimit: tripo3d.Int64(5000),
	Texture:   tripo3d.Bool(true),
})
client.WaitForTask(ctx, modelID, tripo3d.WaitOptions{})

// 2. Verify skeleton compatibility
checkID, _ := client.RigCheck(ctx, tripo3d.RigCheckParams{Input: modelID})
check, _ := client.WaitForTask(ctx, checkID, tripo3d.WaitOptions{})
if !check.Output.IsRiggable() {
	log.Fatal("model is not riggable")
}

// 3. Attach skeleton (Mixamo-compatible bones -> Unity/Unreal ready)
rigID, _ := client.RigModel(ctx, tripo3d.RigModelParams{
	Input:   modelID,
	RigType: tripo3d.String(check.Output.RigType),
	Spec:    tripo3d.String(string(tripo3d.RigSpecMixamo)),
})
client.WaitForTask(ctx, rigID, tripo3d.WaitOptions{})

// 4. Bake preset locomotion animations
animID, _ := client.RetargetAnimation(ctx, tripo3d.RetargetAnimationParams{
	Input:      rigID,
	Animations: []string{string(tripo3d.AnimationIdle), string(tripo3d.AnimationWalk), string(tripo3d.AnimationRun)},
	OutFormat:  tripo3d.String(string(tripo3d.AnimOutFormatGLB)),
})
anim, _ := client.WaitForTask(ctx, animID, tripo3d.WaitOptions{})

fmt.Println("Animated GLB URLs:", anim.Output.ModelURLs)
```

---

## Error handling

```go
import "errors"

taskID, err := client.TextToModel(ctx, params)
if err != nil {
	var apiErr *tripo3d.APIError
	var reqErr *tripo3d.RequestError
	switch {
	case errors.As(err, &apiErr):
		log.Printf("API error %d: %s — %s", apiErr.Code, apiErr.Message, apiErr.Suggestion)
	case errors.As(err, &reqErr):
		log.Printf("transport failure: HTTP %d — %s", reqErr.StatusCode, reqErr.Body)
	default:
		log.Print(err)
	}
}

task, err := client.WaitForTask(ctx, taskID, tripo3d.WaitOptions{Timeout: 5 * time.Minute})
if err != nil {
	var taskErr *tripo3d.TaskError
	var timeoutErr *tripo3d.TimeoutError
	switch {
	case errors.As(err, &taskErr):
		log.Printf("task %s failed: %s", taskErr.Task.TaskID, taskErr.Task.ErrorMsg)
	case errors.As(err, &timeoutErr):
		log.Printf("gave up after %s — task %s", timeoutErr.Timeout, timeoutErr.TaskID)
	default:
		log.Print(err)
	}
}
```

Cancel a poll with `context.WithTimeout` / `context.WithCancel` — `WaitForTask` respects `ctx.Done()` on every iteration.

---

## Constants

```go
tripo3d.TaskStatusSuccess          // "success"
tripo3d.AnimationWalk              // "preset:walk"
tripo3d.RigTypeBiped               // "biped"
tripo3d.RigSpecMixamo              // "mixamo"
tripo3d.ModelVersionH31            // "v3.1-20260211"
tripo3d.ModelVersionP1             // "P1-20260311"
tripo3d.OutputFormatFBX            // "FBX"
```

---

## Running the examples

```bash
export TRIPO_API_KEY="tsk_..."

go run ./examples/text-to-model "a wooden treasure chest"
go run ./examples/image-to-model ./hero.png
go run ./examples/rig-and-animate https://example.com/hero.png
```

## Development

```bash
go build ./...
go vet ./...
go test ./...     # hermetic — uses httptest.Server, no real API key needed
```

Source tree:

```
client.go       # Client, NewClient, task/account/download methods
generation.go   # text/image/multiview -to-model + image generation params & methods
postprocess.go  # models/texture, models/convert, mesh/* params & methods
animation.go    # rig-check, rig, retarget params & methods
http.go         # net/http wrapper with retry + envelope parsing
errors.go       # APIError, RequestError, TaskError, TimeoutError
constants.go    # TaskStatus, Animation, RigType, RigSpec, ModelVersion, …
types.go        # Task, TaskOutput, Balance, FileDescriptor, …
helpers.go      # Bool/String/Int64/Float64 pointer helpers
examples/       # runnable end-to-end demos (one `main` package per example)
client_test.go  # httptest-backed unit tests
```

---

## Reference

- API reference (English): https://developers.tripo3d.com/en/docs/introduction
- API reference (中文): https://developers.tripo3d.com/zh/docs/introduction
- Endpoint details: https://docs.tripo3d.ai/

## License

MIT — see `LICENSE`. Not affiliated with, or endorsed by, VAST AI / Tripo3D.
