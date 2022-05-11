package aws

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/seventv/api/global/instance"
	"github.com/sirupsen/logrus"
)

var (
	DefaultCacheControl = aws.String("public, max-age=15552000")
	AclPublicRead       = aws.String("public-read")
	AclPrivate          = aws.String("private")
)

func NewS3(secretKey, accessToken, region, endpoint string) (instance.AwsS3, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessToken, secretKey, ""),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(true),
		Endpoint:         &endpoint,
	})
	if err != nil {
		return nil, err
	}

	dl := s3manager.NewDownloader(sess)
	up := s3manager.NewUploader(sess)
	s3 := s3.New(sess)

	return &AwsS3Instance{
		sess:       sess,
		downloader: dl,
		uploader:   up,
		s3:         s3,
	}, nil
}

type AwsS3Instance struct {
	sess       *session.Session
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
	s3         *s3.S3
}

func (a *AwsS3Instance) UploadFile(ctx context.Context, bucket, key string, data io.Reader, contentType, acl, cacheControl *string) error {
	result, err := a.uploader.UploadWithContext(ctx, &s3manager.UploadInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         data,
		ACL:          acl,
		ContentType:  contentType,
		CacheControl: cacheControl,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	logrus.Debugf("file uploaded to, %s", result.Location)
	return nil
}

func (a *AwsS3Instance) DownloadFile(ctx context.Context, bucket, key string, file io.WriterAt) error {
	n, err := a.downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}

	logrus.Debugf("%d bytes downloaded from %s %s", n, bucket, key)
	return nil
}
