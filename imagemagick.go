package imagemagick

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"path"
	"strings"

	"cloud.google.com/go/storage"
)

// Global API clients used across function invocations.
var (
	storageClient *storage.Client
)

// resize patterns
var sizes = []int{
	320,
	640,
}

func init() {
	var err error

	storageClient, err = storage.NewClient(context.Background())
	if err != nil {
		log.Fatalf("storage.NewClient: %v", err)
	}
}

type GCSEvent struct {
	Bucket string `json:"bucket"`
	Name   string `json:"name"`
}

func ResizeImage(ctx context.Context, e GCSEvent) error {
	inputBlob := storageClient.Bucket(e.Bucket).Object(e.Name)
	attrs, err := inputBlob.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("Attrs: %v", err)
	}

	// resize済みの画像ならskip
	if attrs.Metadata["isResized"] == "true" {
		return nil
	}

	// 画像でないならskip
	if !strings.HasPrefix(attrs.ContentType, "image/") {
		return nil
	}

	for _, size := range sizes {
		r, err := inputBlob.NewReader(ctx)
		if err != nil {
			return fmt.Errorf("NewReader: %v", err)
		}

		if err := doResize(ctx, e, r, size); err != nil {
			log.Println(err)
			continue
		}
	}
	return nil
}

func doResize(ctx context.Context, e GCSEvent, r io.Reader, size int) error {
	name, err := resizeName(e.Name, size)
	if err != nil {
		return err
	}

	outputBlob := storageClient.Bucket(e.Bucket).Object(name)
	w := outputBlob.NewWriter(ctx)
	defer func() {
		w.Close()
		attrs := storage.ObjectAttrsToUpdate{
			Metadata: map[string]string{
				"isResized": "true",
			},
		}
		if _, err := outputBlob.Update(ctx, attrs); err != nil {
			log.Printf("[error] %s", err)
		}
	}()

	cmd := exec.Command("convert", "-", "-resize", fmt.Sprintf("%dx", size), "-")
	cmd.Stdin = r
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run: %v", err)
	}

	log.Printf("Resized image uploaded to gs://%s/%s", outputBlob.BucketName(), outputBlob.ObjectName())
	return nil
}

func resizeName(name string, size int) (string, error) {
	ext := path.Ext(name)
	basename := name[:len(name)-len(ext)]
	for _, check := range sizes {
		if strings.HasSuffix(basename, fmt.Sprintf("_%d", check)) {
			return "", fmt.Errorf("already resized files")
		}
	}
	return fmt.Sprintf("%s_%d%s", basename, size, ext), nil
}
