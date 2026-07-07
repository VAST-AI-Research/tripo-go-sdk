// Command rig-and-animate runs an end-to-end "game-ready character"
// pipeline:
//
//	image-to-model -> rig-check -> rig -> retarget(walk, idle, run)
//
//	go run ./examples/rig-and-animate https://example.com/hero.png
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	tripo3d "github.com/vast-enterprise/tripo-go-sdk"
)

func main() {
	imageURL := "https://raw.githubusercontent.com/VAST-AI-Research/tripo-python-sdk/master/example.png"
	if len(os.Args) > 1 {
		imageURL = os.Args[1]
	}

	client, err := tripo3d.NewClient(tripo3d.ClientOptions{})
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	// 1. Generate a base 3D model — the P1 line has clean, low-poly topology.
	modelTaskID, err := client.ImageToModel(ctx, tripo3d.ImageToModelParams{
		File:      tripo3d.File(imageURL),
		Model:     tripo3d.String(tripo3d.ModelVersionP1),
		FaceLimit: tripo3d.Int64(5000),
		Texture:   tripo3d.Bool(true),
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := stage(ctx, client, "image-to-model", modelTaskID); err != nil {
		log.Fatal(err)
	}

	// 2. Check whether the model is riggable and what skeleton type fits.
	checkTaskID, err := client.RigCheck(ctx, tripo3d.RigCheckParams{Input: modelTaskID})
	if err != nil {
		log.Fatal(err)
	}
	checkTask, err := stage(ctx, client, "rig-check", checkTaskID)
	if err != nil {
		log.Fatal(err)
	}

	if !checkTask.Output.IsRiggable() {
		log.Fatalf("Model is not riggable (rig_type=%s). Aborting.", checkTask.Output.RigType)
	}
	rigType := checkTask.Output.RigType
	fmt.Printf("  -> riggable=true, rig_type=%s\n", rigType)

	// 3. Attach the skeleton (use Mixamo naming so it drops into Unity/Unreal).
	rigTaskID, err := client.RigModel(ctx, tripo3d.RigModelParams{
		Input:   modelTaskID,
		RigType: tripo3d.String(rigType),
		Spec:    tripo3d.String(string(tripo3d.RigSpecMixamo)),
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := stage(ctx, client, "rig", rigTaskID); err != nil {
		log.Fatal(err)
	}

	// 4. Retarget preset animations.
	animTaskID, err := client.RetargetAnimation(ctx, tripo3d.RetargetAnimationParams{
		Input:      rigTaskID,
		Animations: []string{string(tripo3d.AnimationIdle), string(tripo3d.AnimationWalk), string(tripo3d.AnimationRun)},
		OutFormat:  tripo3d.String(string(tripo3d.AnimOutFormatGLB)),
	})
	if err != nil {
		log.Fatal(err)
	}
	animTask, err := stage(ctx, client, "retarget", animTaskID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nAnimated model URLs:")
	if animTask.Output != nil {
		for i, u := range animTask.Output.ModelURLs {
			fmt.Printf("  [%d] %s\n", i, u)
		}
	}

	downloaded, err := client.DownloadModel(ctx, animTask)
	if err != nil {
		log.Fatal(err)
	}
	if downloaded != nil {
		filename := fmt.Sprintf("character-%s.glb", animTaskID)
		if err := os.WriteFile(filename, downloaded.Data, 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("> saved %s (%d bytes)\n", filename, len(downloaded.Data))
	}
}

func stage(ctx context.Context, client *tripo3d.Client, label, taskID string) (*tripo3d.Task, error) {
	fmt.Printf("\n[%s] task_id=%s\n", label, taskID)
	task, err := client.WaitForTask(ctx, taskID, tripo3d.WaitOptions{
		PollInterval: 2 * time.Second,
		OnProgress: func(t *tripo3d.Task) {
			fmt.Printf("\r  %s — %d%%   ", t.Status, t.Progress)
		},
	})
	fmt.Println()
	return task, err
}
