package instance

import (
	"context"
	"io"
)

type AwsS3 interface {
	UploadFile(ctx context.Context, bucket, key string, data io.Reader, contentType, acl, cacheControl *string) error
	DownloadFile(ctx context.Context, bucket, key string, file io.WriterAt) error
}
