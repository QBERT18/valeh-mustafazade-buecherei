package storage

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Storage is the S3 client (RustFS) plus the bucket the book files live in.
type Storage struct {
	Client *minio.Client
	Bucket string
}

// BookFileKey returns a new, unique object key for a book's file.
// Every upload gets a fresh key, so an existing file is never overwritten.
func BookFileKey(bookID uint, ext string) string {
	return fmt.Sprintf("books/%d/%s%s", bookID, uuid.NewString(), ext)
}

// Conn returns the shared storage client. Like db.Conn it is created on the
// first call and reused afterwards; *minio.Client is safe for concurrent use.
var Conn = sync.OnceValues(Open)

// Open creates a new storage client from the S3_* variables.
// The app uses Conn; Open is for callers that need their own (e.g. tests).
func Open() (*Storage, error) {
	var missing []string
	get := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	endpoint := get("S3_ENDPOINT")
	accessKey := get("S3_ACCESS_KEY")
	secretKey := get("S3_SECRET_KEY")
	bucket := get("S3_BUCKET")
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: os.Getenv("S3_USE_SSL") == "true",
	})
	if err != nil {
		return nil, err
	}

	// Fail fast if the storage is unreachable or the bucket is missing.
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("bucket %q does not exist", bucket)
	}

	return &Storage{Client: client, Bucket: bucket}, nil
}
