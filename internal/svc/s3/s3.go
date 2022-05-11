package s3

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/seventv/api/internal/instance"
)

type Instance struct {
	session    *session.Session
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
	s3         *s3.S3
}

func New(ctx context.Context, o Options) (instance.S3, error) {
	s, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(o.AccessToken, o.SecretKey, ""),
		Region:           aws.String(o.Region),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         aws.String(o.Endpoint),
	})
	if err != nil {
		return nil, err
	}

	return &Instance{
		session:    s,
		downloader: s3manager.NewDownloader(s),
		uploader:   s3manager.NewUploader(s),
		s3:         s3.New(s),
	}, nil
}

func (a *Instance) UploadFile(ctx context.Context, opts *s3manager.UploadInput) error {
	_, err := a.uploader.UploadWithContext(ctx, opts)

	return err
}

func (a *Instance) DownloadFile(ctx context.Context, output io.WriterAt, opts *s3.GetObjectInput) error {
	_, err := a.downloader.DownloadWithContext(ctx, output, opts)

	return err
}
