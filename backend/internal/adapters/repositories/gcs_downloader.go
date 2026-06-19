package repositories

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/lume/backend/internal/core/ports"
	"google.golang.org/api/iterator"
)

type GCSDownloader struct {
	client *storage.Client
	bucket string
	logger *slog.Logger
}

// NewGCSDownloader creates a GCSDownloader. The target bucket is read from the
// GCS_BUCKET environment variable and must be set.
func NewGCSDownloader(logger *slog.Logger) (ports.StateDownloader, error) {
	bucket := os.Getenv("GCS_BUCKET")
	if bucket == "" {
		return nil, fmt.Errorf("GCS_BUCKET environment variable is not set")
	}
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	return &GCSDownloader{client: client, bucket: bucket, logger: logger.With("component", "gcs_downloader")}, nil
}

func (d *GCSDownloader) DownloadState(ctx context.Context, bucketName, objectName string) ([]byte, error) {
	rc, err := d.client.Bucket(bucketName).Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("open gs://%s/%s: %w", bucketName, objectName, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read gs://%s/%s: %w", bucketName, objectName, err)
	}

	d.logger.DebugContext(ctx, "downloaded object", "bucket", bucketName, "object", objectName, "bytes", len(data))
	return data, nil
}

// ListStateObjects returns every object name in the configured bucket
// whose name ends with ".tfstate".
func (d *GCSDownloader) ListStateObjects(ctx context.Context) ([]string, error) {
	it := d.client.Bucket(d.bucket).Objects(ctx, &storage.Query{})

	var (
		objects []string
		scanned int
	)
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list bucket %q: %w", d.bucket, err)
		}
		scanned++
		if strings.HasSuffix(attrs.Name, ".tfstate") {
			objects = append(objects, attrs.Name)
		}
	}
	d.logger.DebugContext(ctx, "listed bucket", "bucket", d.bucket, "scanned", scanned, "tfstate_matches", len(objects))
	return objects, nil
}
