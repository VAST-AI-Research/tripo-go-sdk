// Command text-to-image generates a concept image from a text prompt, waits
// for completion, then saves every URL found in the task output to disk.
//
//	export TRIPO_API_KEY="tsk_..."
//	go run ./examples/text-to-image "a cute red panda holding bamboo"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	tripo3d "github.com/VAST-AI-Research/tripo-go-sdk"
)

func main() {
	prompt := strings.Join(os.Args[1:], " ")
	if prompt == "" {
		prompt = "a cute red panda holding bamboo, studio lighting"
	}
	fmt.Printf("> prompt: %s\n", prompt)

	client, err := tripo3d.NewClient(tripo3d.ClientOptions{})
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	taskID, err := client.TextToImage(ctx, tripo3d.TextToImageParams{Prompt: prompt})
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

	if task.Output == nil {
		fmt.Println("Task succeeded but returned no output.")
		return
	}
	fmt.Printf("> raw output: %s\n", string(task.Output.Raw))

	urls := collectURLs(task.Output.Raw)
	for i, u := range urls {
		if err := download(fmt.Sprintf("tripo-%s-%d%s", taskID, i, guessExt(u)), u); err != nil {
			log.Printf("download failed for %s: %v", u, err)
			continue
		}
	}
}

var urlPattern = regexp.MustCompile(`"(https?://[^"]+)"`)

func collectURLs(raw json.RawMessage) []string {
	matches := urlPattern.FindAllStringSubmatch(string(raw), -1)
	seen := map[string]bool{}
	var urls []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			urls = append(urls, m[1])
		}
	}
	return urls
}

func download(filename, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("> saved %s (%d bytes)\n", filename, len(data))
	return nil
}

func guessExt(url string) string {
	base := path.Ext(strings.SplitN(path.Base(url), "?", 2)[0])
	if base == "" {
		return ".bin"
	}
	return base
}
