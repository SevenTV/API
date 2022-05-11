package instance

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3 interface {
	UploadFile(ctx context.Context, opts *s3manager.UploadInput) error
	DownloadFile(ctx context.Context, output io.WriterAt, opts *s3.GetObjectInput) error
}
