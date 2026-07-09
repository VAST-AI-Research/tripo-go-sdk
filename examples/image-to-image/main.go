// Command image-to-image applies a style/edit transformation to an existing
// image and saves the result to disk.
//
//	# local file
//	go run ./examples/image-to-image ./cat.png "turn it into a watercolor painting"
//	# remote URL
//	go run ./examples/image-to-image https://example.com/cat.jpg "make it look like a pencil sketch"
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
	"path/filepath"
	"regexp"
	"strings"
	"time"

	tripo3d "github.com/vast-enterprise/tripo-go-sdk"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run ./examples/image-to-image <local-file|url> [prompt]")
	}
	input := os.Args[1]
	prompt := strings.Join(os.Args[2:], " ")
	if prompt == "" {
		prompt = "turn it into a watercolor painting"
	}

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

	fmt.Printf("> prompt: %s\n", prompt)
	taskID, err := client.ImageToImage(ctx, tripo3d.ImageToImageParams{
		File:   &file,
		Prompt: tripo3d.String(prompt),
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

	if task.Output == nil {
		fmt.Println("Task succeeded but returned no output.")
		return
	}
	fmt.Printf("> raw output: %s\n", string(task.Output.Raw))

	for i, u := range collectURLs(task.Output.Raw) {
		if err := download(fmt.Sprintf("tripo-%s-%d%s", taskID, i, guessExt(u)), u); err != nil {
			log.Printf("download failed for %s: %v", u, err)
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

func guessContentType(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}
