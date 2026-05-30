package pkg

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type PresignDisposition string

const (
	PresignInline     PresignDisposition = "inline"
	PresignAttachment PresignDisposition = "attachment"
)

type PresignOptions struct {
	Disposition PresignDisposition
	Filename    string
}

type StoreProvider interface {
	EnsureBucket(ctx context.Context) error
	Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error
	DownloadToFile(ctx context.Context, objectKey, destPath string) error
	PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration, opts *PresignOptions) (string, error)
	Remove(ctx context.Context, objectKey string) error
	Bucket() string
}

type minioStore struct {
	client        *minio.Client // internal: upload/download (MINIO_ENDPOINT)
	presignClient *minio.Client // signs URLs for browser host (MINIO_PUBLIC_ENDPOINT)
	bucket        string
}

func NewStoreProvider(endpoint, publicEndpoint, accessKey, secretKey string, useSSL bool, bucket string) (StoreProvider, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("minio endpoint is required")
	}
	if accessKey == "" {
		return nil, fmt.Errorf("minio access key is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("minio secret key is required")
	}
	if bucket == "" {
		return nil, fmt.Errorf("minio bucket is required")
	}

	client, err := newMinioClient(endpoint, useSSL, accessKey, secretKey, nil)
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	presignTarget := strings.TrimSpace(publicEndpoint)
	if presignTarget == "" {
		presignTarget = endpoint
	}

	presignClient := client
	if !endpointsEqual(endpoint, presignTarget, useSSL) {
		internalHost, _, err := parseMinioEndpoint(endpoint, useSSL)
		if err != nil {
			return nil, fmt.Errorf("minio internal endpoint: %w", err)
		}
		presignClient, err = newMinioClient(presignTarget, useSSL, accessKey, secretKey, dialOverrideTransport(internalHost))
		if err != nil {
			return nil, fmt.Errorf("minio presign client: %w", err)
		}
	}

	return &minioStore{
		client:        client,
		presignClient: presignClient,
		bucket:        bucket,
	}, nil
}

func dialOverrideTransport(internalHost string) http.RoundTripper {
	base := http.DefaultTransport.(*http.Transport).Clone()
	internalHost = strings.TrimSpace(internalHost)
	base.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			port = "9000"
		}
		target := internalHost
		if _, _, err := net.SplitHostPort(target); err != nil {
			target = net.JoinHostPort(target, port)
		}
		var d net.Dialer
		return d.DialContext(ctx, network, target)
	}
	return base
}

func newMinioClient(endpoint string, defaultSecure bool, accessKey, secretKey string, transport http.RoundTripper) (*minio.Client, error) {
	host, secure, err := parseMinioEndpoint(endpoint, defaultSecure)
	if err != nil {
		return nil, err
	}
	opts := &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       secure,
		BucketLookup: minio.BucketLookupPath,
	}
	if transport != nil {
		opts.Transport = transport
	}
	return minio.New(host, opts)
}

func parseMinioEndpoint(raw string, defaultSecure bool) (host string, secure bool, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, fmt.Errorf("empty endpoint")
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", false, err
		}
		if u.Host == "" {
			return "", false, fmt.Errorf("invalid endpoint %q", raw)
		}
		return u.Host, u.Scheme == "https", nil
	}
	return raw, defaultSecure, nil
}

func endpointsEqual(a, b string, defaultSecure bool) bool {
	ha, sa, errA := parseMinioEndpoint(a, defaultSecure)
	hb, sb, errB := parseMinioEndpoint(b, defaultSecure)
	if errA != nil || errB != nil {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}
	return ha == hb && sa == sb
}

func (s *minioStore) Bucket() string {
	return s.bucket
}

func (s *minioStore) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		exists, err := s.client.BucketExists(ctx, s.bucket)
		if err == nil && exists {
			return nil
		}
		return err
	}
	return nil
}

func (s *minioStore) Put(ctx context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	if objectKey == "" {
		return fmt.Errorf("objectKey is required")
	}
	_, err := s.client.PutObject(ctx, s.bucket, objectKey, reader, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func contentDisposition(opts *PresignOptions) string {
	disposition := PresignInline
	filename := ""
	if opts != nil {
		if opts.Disposition != "" {
			disposition = opts.Disposition
		}
		filename = strings.TrimSpace(opts.Filename)
	}
	if filename == "" {
		return string(disposition)
	}
	safe := strings.ReplaceAll(filename, `"`, `\"`)
	return fmt.Sprintf(`%s; filename="%s"`, disposition, safe)
}

// PresignedGetURL returns a GET URL signed on MINIO_PUBLIC_ENDPOINT (browser must reach that host).
func (s *minioStore) PresignedGetURL(ctx context.Context, objectKey string, expiry time.Duration, opts *PresignOptions) (string, error) {
	if objectKey == "" {
		return "", fmt.Errorf("objectKey is required")
	}
	if expiry <= 0 {
		expiry = time.Hour
	}

	_, err := s.client.StatObject(ctx, s.bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("object not found: %s/%s: %w", s.bucket, objectKey, err)
	}

	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", contentDisposition(opts))

	u, err := s.presignClient.PresignedGetObject(ctx, s.bucket, objectKey, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *minioStore) Remove(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("objectKey is required")
	}
	return s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{})
}

func (s *minioStore) DownloadToFile(ctx context.Context, objectKey, destPath string) error {
	if objectKey == "" {
		return fmt.Errorf("objectKey is required")
	}
	if destPath == "" {
		return fmt.Errorf("destPath is required")
	}
	return s.client.FGetObject(ctx, s.bucket, objectKey, destPath, minio.GetObjectOptions{})
}
