package repositories

import (
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/lume/backend/internal/core/ports"
)

type GCSDownloader struct {
	client *storage.Client
}

func NewGCSDownloader() (ports.StateDownloader, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSDownloader{client: client}, nil
}

func (d *GCSDownloader) DownloadState(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	rc, err := d.client.Bucket(bucketName).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return data, nil
}
