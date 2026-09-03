package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ObjectStore is blob storage (S3/MinIO).
type ObjectStore interface {
	// Presign returns a time-limited URL the client can use to PUT or GET the key directly.
	Presign(ctx context.Context, key, method string, ttl time.Duration) (string, error)
	Get(ctx context.Context, key string) ([]byte, error)
	// GetFirstNBytes returns the first n bytes of the object (used for content sniffing).
	GetFirstNBytes(ctx context.Context, key string, n int64) ([]byte, error)
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Head(ctx context.Context, key string) (size int64, contentType string, err error)
	Delete(ctx context.Context, key string) error
}

// MinioStore implements ObjectStore against MinIO/S3.
// It holds TWO clients:
//   - `ops` talks to the INTERNAL endpoint (e.g. minio:9000) for real reads/writes from inside the cluster.
//   - `presign` is configured with the PUBLIC endpoint (e.g. localhost:9000) so the URLs it signs point
//     at an address the browser/Postman client can actually reach
type MinioStore struct {
	ops     *minio.Client
	presign *minio.Client
	bucket  string
}

func NewMinioStore(internalEndpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStore, error) {
	creds := credentials.NewStaticV4(accessKey, secretKey, "")
	// Region is set explicitly so presigning never triggers a GetBucketLocation
	// network call — presigning must stay a purely local computation, especially
	// on the presign client whose public endpoint isn't reachable from here.
	opts := func(endpoint string) *minio.Options {
		return &minio.Options{Creds: creds, Secure: useSSL, Region: "us-east-1"}
	}
	ops, err := minio.New(internalEndpoint, opts(internalEndpoint))
	if err != nil {
		return nil, err
	}
	presign, err := minio.New(publicEndpoint, opts(publicEndpoint))
	if err != nil {
		return nil, err
	}
	return &MinioStore{ops: ops, presign: presign, bucket: bucket}, nil
}

// EnsureBucket creates the bucket if it doesn't already exist. Called at startup.
func (store *MinioStore) EnsureBucket(ctx context.Context) error {
	exists, err := store.ops.BucketExists(ctx, store.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return store.ops.MakeBucket(ctx, store.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (store *MinioStore) Presign(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	switch method {
	case "PUT":
		u, err := store.presign.PresignedPutObject(ctx, store.bucket, key, ttl)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	case "GET":
		u, err := store.presign.PresignedGetObject(ctx, store.bucket, key, ttl, nil)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported presign method: %s", method)
	}
}

func (store *MinioStore) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := store.ops.GetObject(ctx, store.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (store *MinioStore) GetFirstNBytes(ctx context.Context, key string, n int64) ([]byte, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(0, n-1); err != nil {
		return nil, err
	}
	obj, err := store.ops.GetObject(ctx, store.bucket, key, opts)
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (store *MinioStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := store.ops.PutObject(ctx, store.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (store *MinioStore) Head(ctx context.Context, key string) (int64, string, error) {
	info, err := store.ops.StatObject(ctx, store.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, "", err
	}
	return info.Size, info.ContentType, nil
}

func (store *MinioStore) Delete(ctx context.Context, key string) error {
	return store.ops.RemoveObject(ctx, store.bucket, key, minio.RemoveObjectOptions{})
}
