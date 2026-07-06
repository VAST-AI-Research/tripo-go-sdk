// Command image-to-model converts a local or remote image into a 3D model.
//
//	# local file
//	go run ./examples/image-to-model ./cat.png
//	# remote URL
//	go run ./examples/image-to-model https://example.com/cat.jpg
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tripo3d "github.com/your-org/tripo3d-sdk-go"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./examples/image-to-model <local-file|url>")
	}
	input := os.Args[1]

	client, err := tripo3d.NewClient(tripo3d.ClientOptions{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	var file tripo3d.FileDescriptor
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		file = tripo3d.File(input)
	} else {
		data, err := os.ReadFile(input)
		if err != nil {
			log.Fatal(err)
		}
		uploaded, err := client.UploadFile(ctx, data, filepath.Base(input), guessContentType(input))
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("> uploaded, file_token=%s\n", uploaded.FileToken)
		file = tripo3d.File(uploaded.FileToken)
	}

	taskID, err := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
		File:             file,
		Model:            tripo3d.String(tripo3d.ModelVersionH31),
		Texture:          tripo3d.Bool(true),
		PBR:              tripo3d.Bool(true),
		TextureAlignment: tripo3d.String("original_image"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("> submitted, task_id=%s\n", taskID)

	task, err := client.WaitForTask(ctx, taskID, tripo3d.WaitOptions{
		PollInterval: 2 * time.Second,
		OnProgress: func(t *tripo3d.Task) {
			fmt.Printf("\r  %s — %d%%   ", t.Status, t.Progress)
		},
	})
	fmt.Println()
	if err != nil {
		log.Fatal(err)
	}

	downloaded, err := client.DownloadModel(ctx, task)
	if err != nil {
		log.Fatal(err)
	}
	if downloaded != nil {
		filename := fmt.Sprintf("tripo-%s.glb", taskID)
		if err := os.WriteFile(filename, downloaded.Data, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("> saved %s (%d bytes)\n", filename, len(downloaded.Data))
	}
}

func guessContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	default:
		return "application/octet-stream"
	}
}
