// Command text-to-model generates a 3D model from a text prompt, waits for
// completion, then saves the resulting GLB to disk.
//
//	export TRIPO_API_KEY="tsk_..."
//	go run ./examples/text-to-model "a cute red panda"
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tripo3d "github.com/your-org/tripo3d-sdk-go"
)

func main() {
	prompt := strings.Join(os.Args[1:], " ")
	if prompt == "" {
		prompt = "a cute red panda holding bamboo"
	}
	fmt.Printf("> prompt: %s\n", prompt)

	client, err := tripo3d.NewClient(tripo3d.ClientOptions{}) // reads TRIPO_API_KEY
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	taskID, err := client.TextToModel(ctx, tripo3d.TextToModelParams{
		Prompt:         prompt,
		Model:          tripo3d.String(tripo3d.ModelVersionH31),
		Texture:        tripo3d.Bool(true),
		PBR:            tripo3d.Bool(true),
		TextureQuality: tripo3d.String("detailed"),
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
	if downloaded == nil {
		fmt.Printf("Task succeeded but no model URL was returned: %+v\n", task.Output)
		return
	}

	filename := fmt.Sprintf("tripo-%s.glb", taskID)
	if err := os.WriteFile(filename, downloaded.Data, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("> saved %s (%d bytes)\n", filename, len(downloaded.Data))
}
