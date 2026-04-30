package repositories

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/lume/backend/internal/core/ports"
	"google.golang.org/api/iterator"
)

type GCSDownloader struct {
	client *storage.Client
	bucket string
}

// NewGCSDownloader creates a GCSDownloader. The target bucket is read from the
// GCS_BUCKET environment variable and must be set.
func NewGCSDownloader() (ports.StateDownloader, error) {
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("GCS_BUCKET environment variable is not set")
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	return &GCSDownloader{client: client, bucket: bucket}, nil
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

// ListStateObjects returns every object name in the configured bucket
// whose name ends with ".tfstate".
func (d *GCSDownloader) ListStateObjects(ctx context.Context) ([]string, error) {
	it := d.client.Bucket(d.bucket).Objects(ctx, &storage.Query{})

	var objects []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(attrs.Name, ".tfstate") {
			objects = append(objects, attrs.Name)
		}
	}
	return objects, nil
}
