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

// ObjectStore is blob storage (S3/MinIO). Bytes are keyed, not path-based, and
// clients upload/download directly via presigned URLs so this service never
// buffers a file.
type ObjectStore interface {
	// Presign returns a time-limited URL the client can use to PUT or GET the key directly.
	Presign(ctx context.Context, key, method string, ttl time.Duration) (string, error)
	Get(ctx context.Context, key string) ([]byte, error)
	// GetRange returns the first n bytes of the object (used for content sniffing).
	GetRange(ctx context.Context, key string, n int64) ([]byte, error)
	Put(ctx context.Context, key string, data []byte, contentType string) error
	Head(ctx context.Context, key string) (size int64, contentType string, err error)
	Delete(ctx context.Context, key string) error
}

// MinioStore implements ObjectStore against MinIO/S3.
//
// It holds TWO clients on purpose. `ops` talks to the INTERNAL endpoint
// (e.g. minio:9000) for real reads/writes from inside the cluster. `presign`
// is configured with the PUBLIC endpoint (e.g. localhost:9000) so the URLs it
// signs point at an address the browser/Postman client can actually reach —
// and because the host is part of the S3 signature, you can't just rewrite it
// after the fact.
type MinioStore struct {
	ops     *minio.Client
	presign *minio.Client
	bucket  string
}

func NewMinioStore(internalEndpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*MinioStore, error) {
	creds := credentials.NewStaticV4(accessKey, secretKey, "")
	ops, err := minio.New(internalEndpoint, &minio.Options{Creds: creds, Secure: useSSL})
	if err != nil {
		return nil, err
	}
	presign, err := minio.New(publicEndpoint, &minio.Options{Creds: creds, Secure: useSSL})
	if err != nil {
		return nil, err
	}
	return &MinioStore{ops: ops, presign: presign, bucket: bucket}, nil
}

// EnsureBucket creates the bucket if it doesn't already exist. Called at startup.
func (s *MinioStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.ops.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.ops.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (s *MinioStore) Presign(ctx context.Context, key, method string, ttl time.Duration) (string, error) {
	switch method {
	case "PUT":
		u, err := s.presign.PresignedPutObject(ctx, s.bucket, key, ttl)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	case "GET":
		u, err := s.presign.PresignedGetObject(ctx, s.bucket, key, ttl, nil)
		if err != nil {
			return "", err
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("unsupported presign method: %s", method)
	}
}

func (s *MinioStore) Get(ctx context.Context, key string) ([]byte, error) {
	obj, err := s.ops.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (s *MinioStore) GetRange(ctx context.Context, key string, n int64) ([]byte, error) {
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(0, n-1); err != nil {
		return nil, err
	}
	obj, err := s.ops.GetObject(ctx, s.bucket, key, opts)
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}

func (s *MinioStore) Put(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := s.ops.PutObject(ctx, s.bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *MinioStore) Head(ctx context.Context, key string) (int64, string, error) {
	info, err := s.ops.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, "", err
	}
	return info.Size, info.ContentType, nil
}

func (s *MinioStore) Delete(ctx context.Context, key string) error {
	return s.ops.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
