# tripo3d-sdk（Go 版）

[English](./README.md) · **简体中文**

一个非官方的 **Go SDK**，用于访问 [Tripo3D v3 API](https://developers.tripo3d.com/zh/docs/introduction) —— 覆盖 AI 3D 生成的完整能力：文生 3D、图生 3D、多视角生 3D、重贴图、网格编辑、自动绑骨与动画重定向。

- **零第三方依赖** —— 完全基于 `net/http` 与标准库构建。
- 所有方法都接受 `context.Context`，天然支持取消与超时。
- 对瞬态网络错误 / 5xx 自动重试，支持 `Retry-After`。
- 强类型的错误值（`*APIError`、`*TaskError`、`*TimeoutError`、`*RequestError`），可配合 `errors.As` 使用。
- `WaitForTask` 轮询器，支持进度回调。
- 与 [`tripo3d-sdk-js`](../tripo3d-sdk-js)、[`tripo3d-sdk-rust`](../tripo3d-sdk-rust) 是同源姊妹 SDK —— API 能力一致，各自遵循语言惯例。

> Base URL：`https://openapi.tripo3d.com/v3` —— 本 SDK 面向 **v3** REST 接口，**不是**旧的 `/v2/openapi/task` 接口。

---

## 安装

```bash
go get github.com/vast-enterprise/tripo-go-sdk
```

如果仓库是私有的，需要让 Go 通过 SSH 拉取，并把它标记为私有模块以跳过公共校验和数据库：

```bash
export GOPRIVATE=github.com/vast-enterprise/*
git config --global url."git@github.com:".insteadOf "https://github.com/"
```

如果是基于本地代码进行开发调试，可以改用 `replace` 指令：

```bash
go mod edit -replace github.com/vast-enterprise/tripo-go-sdk=../tripo3d-sdk-go
go mod tidy
```

先在 [Tripo 控制台](https://platform.tripo3d.ai/) 创建 API Key 并导出：

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

	tripo3d "github.com/vast-enterprise/tripo-go-sdk"
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
	BaseURL:    "",           // 默认：https://openapi.tripo3d.com/v3
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

任何 `FileDescriptor`（或 `*FileDescriptor`）字段都可以用 `tripo3d.File(...)` 构建，它会自动判断是 URL 还是裸 file_token：

```go
f1 := tripo3d.File("https://example.com/hero.png")   // -> FileDescriptor{URL: "..."}
f2 := tripo3d.File("8f2a4c...")                       // -> FileDescriptor{FileToken: "..."}
f3 := tripo3d.FileDescriptor{Object: &tripo3d.ObjectRef{Bucket: "tripo-data", Key: "uploads/abc.png"}}
```

上传本地文件获取 `file_token`：

```go
data, _ := os.ReadFile("./hero.png")
uploaded, err := client.UploadFile(ctx, data, "hero.png", "image/png")

taskID, err := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	File: tripo3d.File(uploaded.FileToken),
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
// 1. 图生 3D（P1 系列低面拓扑，游戏/移动端友好）
modelID, _ := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
	File:      tripo3d.File("https://example.com/hero.png"),
	Model:     tripo3d.String(tripo3d.ModelVersionP1),
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
		log.Printf("任务 %s 失败：%s", taskErr.Task.TaskID, taskErr.Task.ErrorMsg)
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

## 常量枚举

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

- API 文档（英文）：https://developers.tripo3d.com/en/docs/introduction
- API 文档（中文）：https://developers.tripo3d.com/zh/docs/introduction
- 每个端点参数细节：https://docs.tripo3d.ai/

## 许可协议

MIT —— 见 `LICENSE`。本项目与 VAST AI / Tripo3D 官方无隶属关系。
