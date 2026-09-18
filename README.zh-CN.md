# tripo3d-sdk（Go 版）

[English](./README.md) · **简体中文**

Tripo 官方 **Go SDK**，用于访问 [Tripo3D v3 API](https://developers.tripo3d.com/zh/docs/introduction) —— 覆盖 AI 3D 生成的完整能力：文生 3D、图生 3D、多视角生 3D、重贴图、网格编辑、自动绑骨与动画重定向。

- **零第三方依赖** —— 完全基于 `net/http` 与标准库构建。
- 所有方法都接受 `context.Context`，天然支持取消与超时。
- 对瞬态网络错误 / 5xx 自动重试，支持 `Retry-After`。
- 强类型的错误值（`*APIError`、`*TaskError`、`*TimeoutError`、`*RequestError`），可配合 `errors.As` 使用。
- `WaitForTask` 轮询器，支持进度回调。
- 与 [`tripo3d-sdk-js`](../tripo3d-sdk-js)、[`tripo3d-sdk-rust`](../tripo3d-sdk-rust) 是同源姊妹 SDK —— API 能力一致，各自遵循语言惯例。

> 国内 Base URL：`https://openapi.tripo3d.com/v3`  
> 海外 Base URL：`https://openapi.tripo3d.ai/v3`  
> 本 SDK 面向 **v3** REST 接口，**不是**旧的 `/v2/openapi/task` 接口。  
> 可通过 `BaseURL` 选择区域（见[客户端参数](#客户端参数)）。

---

## 安装

```bash
go get github.com/VAST-AI-Research/tripo-go-sdk
```

如果仓库是私有的，需要让 Go 通过 SSH 拉取，并把它标记为私有模块以跳过公共校验和数据库：

```bash
export GOPRIVATE=github.com/VAST-AI-Research/*
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

如果是基于本地代码进行开发调试，可以改用 `replace` 指令：

```bash
go mod edit -replace github.com/VAST-AI-Research/tripo-go-sdk=../tripo3d-sdk-go
go mod tidy
```

先在 [Tripo 控制台](https://platform.tripo3d.com/) 创建 API Key 并导出（海外请使用 [platform.tripo3d.ai](https://platform.tripo3d.ai/)）：

```bash
export TRIPO_API_KEY="tsk_..."
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	tripo3d "github.com/VAST-AI-Research/tripo-go-sdk"
)

func main() {
	client, err := tripo3d.NewClient(tripo3d.ClientOptions{}) // 默认读取 TRIPO_API_KEY
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	taskID, err := client.TextToModel(ctx, tripo3d.TextToModelParams{
		Prompt:         "一只可爱的红熊猫，抱着一根竹子",
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

	fmt.Println("模型下载地址：", task.PrimaryModelURL())
}
```

> ⚠️ 模型 URL 会在任务成功约 **5 分钟后过期**，请及时下载。可用 `client.DownloadModel(ctx, task)`。

---

## 客户端参数

```go
tripo3d.NewClient(tripo3d.ClientOptions{
	APIKey:     "",           // 默认读取 TRIPO_API_KEY 环境变量
	BaseURL:    "",           // 国内：https://openapi.tripo3d.com/v3 · 海外：https://openapi.tripo3d.ai/v3
	HTTPClient: nil,          // 默认：&http.Client{}
	Timeout:    0,            // 单次请求超时，默认 60 秒
	Retries:    0,            // 5xx / 网络错误的额外重试次数；0 = 默认值(2)，-1 = 禁用重试
	UserAgent:  "",
})
```

---

## API 一览

所有生成类方法都返回一个 `task_id`（`string`），随后可以用 `WaitForTask` 等待任务终态结果。

### 3D 生成

| 方法 | 端点 | 说明 |
| --- | --- | --- |
| `TextToModel(ctx, params)` | `POST /generation/text-to-model` | 文本 → 3D 模型 |
| `ImageToModel(ctx, params)` | `POST /generation/image-to-model` | 单图 → 3D 模型 |
| `MultiviewToModel(ctx, params)` | `POST /generation/multiview-to-model` | 4 视角 `[front, left, back, right]` → 3D 模型 |
| `TextToImage(ctx, params)` | `POST /generation/text-to-image` | 文生概念图 |
| `ImageToImage(ctx, params)` | `POST /generation/image-to-image` | 图片风格转换 / 编辑 |
| `ImageToMultiview(ctx, params)` | `POST /generation/image-to-multiview` | 单图 → 4 视角图集 |
| `EditMultiview(ctx, params)` | `POST /generation/edit-multiview` | 编辑已生成的多视角图集 |

### 模型后处理

| 方法 | 端点 | 说明 |
| --- | --- | --- |
| `TextureModel(ctx, params)` | `POST /models/texture` | 为已有模型重新贴图 |
| `ConvertModel(ctx, params)` | `POST /models/convert` | 转换为 GLTF / FBX / OBJ / STL / USDZ / 3MF |
| `SegmentMesh(ctx, params)` | `POST /mesh/segment` | 语义分割 |
| `CompleteMesh(ctx, params)` | `POST /mesh/complete` | 网格补全 / 修复 |
| `DecimateMesh(ctx, params)` | `POST /mesh/decimate` | 减面 / 重拓扑 |

### 动画

| 方法 | 端点 | 说明 |
| --- | --- | --- |
| `RigCheck(ctx, params)` | `POST /animations/rig-check` | 检查模型是否可绑骨、推荐骨骼类型 |
| `RigModel(ctx, params)` | `POST /animations/rig` | 自动绑骨 |
| `RetargetAnimation(ctx, params)` | `POST /animations/retarget` | 应用预设动画 |

### 工具

| 方法 | 端点 | 说明 |
| --- | --- | --- |
| `GetTask(ctx, taskID)` | `GET /tasks/{task_id}` | 查询单个任务 |
| `ListTasks(ctx, taskIDs)` | `POST /tasks/list` | 批量查询任务 |
| `WaitForTask(ctx, taskID, opts)` | — | 轮询直到任务进入终态 |
| `UploadFile(ctx, data, filename, contentType)` | `POST /files` | 上传文件，返回 `file_token` |
| `GetBalance(ctx)` | `GET /account/balance` | 查询账户余额 |
| `DownloadModel(ctx, task)` | — | 把任务的主模型 URL 下载为 `[]byte` |

---

## 传入图片 / 文件

所有接收图片或模型的接口都通过 `FileDescriptor` 类型的 `Input` 字段传入。用 `tripo3d.File(...)` 构建，具体是哪种引用由服务端自动推断 —— 公开 URL、`file_token`，或是需要复用其产物的上游任务 `task_id`：

```go
tripo3d.File("https://example.com/hero.png") // 公开 URL
tripo3d.File("8f2a4c...")                    // UploadFile 返回的 file_token
tripo3d.File(previousTaskID)                 // 复用上游任务的产物
```

如果不想依赖推断，也可以用显式构造函数：

```go
tripo3d.FileURL("https://example.com/hero.png")
tripo3d.FileToken("8f2a4c...")
tripo3d.FileDescriptor{Object: &tripo3d.ObjectRef{Bucket: "tripo-data", Key: "uploads/abc.png"}}
```

上传本地文件获取 `file_token`：

```go
data, _ := os.ReadFile("./hero.png")
uploaded, err := client.UploadFile(ctx, data, "hero.png", "image/png")

taskID, err := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	Input: tripo3d.File(uploaded.FileToken),
})
```

任务串联无需下载再上传，直接把上游 `task_id` 传进去即可：

```go
imageID, _ := client.TextToImage(ctx, tripo3d.TextToImageParams{
	Prompt: "一个低面数木质藏宝箱",
	Model:  tripo3d.String(tripo3d.ImageModelSeedreamV5),
})
client.WaitForTask(ctx, imageID, tripo3d.WaitOptions{})

modelID, _ := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	Input: tripo3d.File(imageID),
	Model: tripo3d.String(tripo3d.ModelVersionP2),
})
```

### 可选字段与指针辅助函数

可选的标量字段都是指针类型（这样 SDK 才能区分"未设置"与"零值"）。用 `tripo3d.Bool`、`tripo3d.String`、`tripo3d.Int64`、`tripo3d.Float64` 辅助函数来填充它们：

```go
tripo3d.TextToModelParams{
	Prompt:  "a cat",
	Texture: tripo3d.Bool(true),
	Model:   tripo3d.String(tripo3d.ModelVersionH31),
}
```

每个 `*Params` 结构体都有一个 `Extra map[string]interface{}` 字段，用于透传 SDK 尚未建模的新字段。

---

## 端到端流水线：游戏就绪角色

```go
// 1. 图生 3D（P 系列低面拓扑，游戏/移动端友好）
modelID, _ := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	Input:     tripo3d.File("https://example.com/hero.png"),
	Model:     tripo3d.String(tripo3d.ModelVersionP2),
	FaceLimit: tripo3d.Int64(5000),
	Texture:   tripo3d.Bool(true),
})
client.WaitForTask(ctx, modelID, tripo3d.WaitOptions{})

// 2. 检查是否可绑骨
checkID, _ := client.RigCheck(ctx, tripo3d.RigCheckParams{Input: modelID})
check, _ := client.WaitForTask(ctx, checkID, tripo3d.WaitOptions{})
if !check.Output.IsRiggable() {
	log.Fatal("该模型不可绑骨")
}

// 3. 自动绑骨（Mixamo 命名 → 可直接导入 Unity / Unreal）
rigID, _ := client.RigModel(ctx, tripo3d.RigModelParams{
	Input:   modelID,
	RigType: tripo3d.String(check.Output.RigType),
	Spec:    tripo3d.String(string(tripo3d.RigSpecMixamo)),
})
client.WaitForTask(ctx, rigID, tripo3d.WaitOptions{})

// 4. 烘焙预设动画
animID, _ := client.RetargetAnimation(ctx, tripo3d.RetargetAnimationParams{
	Input:      rigID,
	Animations: []string{string(tripo3d.AnimationIdle), string(tripo3d.AnimationWalk), string(tripo3d.AnimationRun)},
	OutFormat:  tripo3d.String(string(tripo3d.AnimOutFormatGLB)),
})
anim, _ := client.WaitForTask(ctx, animID, tripo3d.WaitOptions{})

fmt.Println("带动画的 GLB URLs：", anim.Output.ModelURLs)
```

**开发者小贴士：**
- 绑骨前**务必**先调 `RigCheck`，可以拿到推荐的 `RigType` 并避免直接调 `RigModel` 失败。
- 轮询频率建议 **2s** 一次，**不要超过 1 次/秒**，否则可能被限流。
- Unity / Unreal 使用 `RigSpecMixamo`；自建流水线用 `RigSpecTripo`。
- 一次 `RetargetAnimation` 最多传 **5 个** 预设动画（超过会在本地直接报错，不会发出请求）。

---

## 错误处理

```go
import "errors"

taskID, err := client.TextToModel(ctx, params)
if err != nil {
	var apiErr *tripo3d.APIError
	var reqErr *tripo3d.RequestError
	switch {
	case errors.As(err, &apiErr):
		log.Printf("API 错误 %d：%s — %s", apiErr.Code, apiErr.Message, apiErr.Suggestion)
	case errors.As(err, &reqErr):
		log.Printf("传输错误：HTTP %d — %s", reqErr.StatusCode, reqErr.Body)
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
		log.Printf("任务 %s 失败：%s", taskErr.Task.TaskID, taskErr.Task.ErrorMessage)
	case errors.As(err, &timeoutErr):
		log.Printf("任务 %s 在 %s 内未完成", timeoutErr.TaskID, timeoutErr.Timeout)
	default:
		log.Print(err)
	}
}
```

用 `context.WithTimeout` / `context.WithCancel` 取消轮询 —— `WaitForTask` 在每次循环都会检查 `ctx.Done()`。

### 常见错误码

| code | 含义 | 建议 |
| --- | --- | --- |
| `0` | 成功 | — |
| `1xxx` | 参数 / 认证错误 | 检查 `APIKey` 与请求参数 |
| `2010` | 积分不足 | 前往控制台充值 |
| `429` | 请求过于频繁 | 降低并发或延长轮询间隔（SDK 会自动退避重试） |
| `5xx` | 服务端错误 | SDK 会自动重试，仍失败请稍后再试 |

---

### 重试与重复提交

任务创建接口按提交次数计费,因此 SDK 绝不会重放一个服务端可能已经收下的请求。只有在**能证明请求从未被处理**时才重试——连接被拒绝、DNS 解析失败,或服务端明确返回 `429` / `503`。而语义不明的失败(发送途中连接被重置、超时、`500` / `502` / `504`)会让非幂等请求立即失败;幂等的读请求则照常重试。

任务创建失败且状态不明时,错误上会带标记,让你能区分"确定失败"和"状态不确定":

```go
taskID, err := client.ImageToImage(ctx, params)
if err != nil {
    var reqErr *tripo3d.RequestError
    if errors.As(err, &reqErr) && reqErr.Indeterminate {
        // The submission may have gone through. Check ListTasks rather
        // than resubmitting.
        return err
    }
    // Definitely failed; safe to retry yourself.
}
```

此时应先去任务列表核对,不要盲目重发——盲目重试正是重复扣费的根源。

## 常量枚举

```go
tripo3d.TaskStatusSuccess              // "success"
tripo3d.AnimationWalk                  // "preset:walk"
tripo3d.RigTypeBiped                   // "biped"
tripo3d.RigSpecMixamo                  // "mixamo"
tripo3d.ModelVersionH31                // "v3.1-20260211"
tripo3d.ModelVersionP2                 // "P2-20260801"
tripo3d.ImageModelSeedreamV5           // "seedream_v5"
tripo3d.ImageModelChatImage25Sunburst  // "chat_image_2.5_sunburst"
tripo3d.OutputFormatFBX                // "FBX"
```

### 3D 生成模型

| 常量 | 取值 | 说明 |
| --- | --- | --- |
| `ModelVersionH31` | `v3.1-20260211` | 最新，质量最佳（默认） |
| `ModelVersionH30` | `v3.0-20250812` | 稳定版，支持高级特性 |
| `ModelVersionH25` | `v2.5-20250123` | 旧版本，不支持 `GeometryQuality` |
| `ModelVersionP1` | `P1-20260311` | 低面数，干净拓扑 |
| `ModelVersionP2` | `P2-20260801` | 新一代 P 系列，支持四边面输出。preview 版 |

P 系列中只有 `ModelVersionP2` 支持 `Quad`，传给 `ModelVersionP1` 会返回 `400`。P1 同样不支持 `SmartLowPoly`、`GenerateParts` 和 `GeometryQuality`。

另外，开启 `Quad` 会把输出格式强制为 **FBX** 而非 GLB，所以请从 `ModelURL` 推导扩展名，不要假定是 `.glb` —— 用 `downloaded.Filename(name)` 即可。

### 生图模型

用于 `TextToImage` 与 `ImageToImage`。

| 常量 | 取值 | 说明 |
| --- | --- | --- |
| `ImageModelSeedreamV5` | `seedream_v5` | 最强编辑、风格迁移与多图融合 |
| `ImageModelBanana` | `banana` | 快速 |
| `ImageModelBananaPro` | `banana_pro` | 更高质量 |
| `ImageModelBanana2` | `banana2` | 最新快速选项 |
| `ImageModelChatImage2` | `chat_image_2` | 质量最佳 |
| `ImageModelChatImage25Flare` | `chat_image_2.5_flare` | 2.5 系列速度档 |
| `ImageModelChatImage25Sunburst` | `chat_image_2.5_sunburst` | 2.5 系列精修档 |

部分参数是分模型的：`Quality` 仅 `chat_image_2` 和两个 2.5 模型支持（其它模型传入会直接报错），`Background` 仅两个 2.5 模型支持，`AspectRatio` 仅 banana 系列支持 —— seedream 和 chat_image 请改用 `Size` 控制出图尺寸。

`chat_image_1` 与 `chat_image_1.5` 已被有意移除：它们将分别于 2026-10-23 和 2026-12-01 下线。迁移期间如果仍需使用，可通过 `Extra` 透传。

---

## 运行示例

```bash
export TRIPO_API_KEY="tsk_..."

go run ./examples/text-to-model "一个木质藏宝箱"
go run ./examples/image-to-model ./hero.png
go run ./examples/rig-and-animate https://example.com/hero.png
```

## 开发

```bash
go build ./...
go vet ./...
go test ./...     # 完全 hermetic —— 用 httptest.Server 模拟 HTTP 层，无需真实 API Key
```

源码结构：

```
client.go       # Client、NewClient、任务/账户/下载相关方法
generation.go   # 文本/图片/多视角 -to-model + 图像生成参数与方法
postprocess.go  # models/texture、models/convert、mesh/* 参数与方法
animation.go    # rig-check、rig、retarget 参数与方法
http.go         # net/http 封装（重试 + envelope 解析）
errors.go       # APIError、RequestError、TaskError、TimeoutError
constants.go    # TaskStatus、Animation、RigType、RigSpec、ModelVersion……
types.go        # Task、TaskOutput、Balance、FileDescriptor……
helpers.go      # Bool/String/Int64/Float64 指针辅助函数
examples/       # 端到端示例（每个示例是独立的 main 包）
client_test.go  # 基于 httptest 的单元测试
```

---

## 相关链接

- API 文档：https://developers.tripo3d.com/zh/docs/introduction
- API 端点（国内）：`https://openapi.tripo3d.com/v3`
- API 端点（海外）：`https://openapi.tripo3d.ai/v3`

## 许可协议

MIT —— 见 `LICENSE`。
