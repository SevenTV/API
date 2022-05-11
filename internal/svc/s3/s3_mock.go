package s3

import (
	"context"
	"errors"
	"io"
	"io/ioutil"

	"github.com/SevenTV/Common/sync_map"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/seventv/api/internal/instance"
)

type MockInstance struct {
	files *sync_map.Map[string, *sync_map.Map[string, []byte]]
}

func NewMock(ctx context.Context, files map[string]map[string][]byte) (instance.S3, error) {
	mp := &sync_map.Map[string, *sync_map.Map[string, []byte]]{}
	for k, v := range files {
		mp.Store(k, sync_map.FromStdMap(v))
	}

	return &MockInstance{
		files: mp,
	}, nil
}

func (a *MockInstance) UploadFile(ctx context.Context, opts *s3manager.UploadInput) error {
	data, err := ioutil.ReadAll(opts.Body)
	if err != nil {
		return err
	}
	if opts.Bucket == nil {
		return errors.New(s3.ErrCodeNoSuchBucket)
	}
	if opts.Key == nil {
		return errors.New(s3.ErrCodeNoSuchKey)
	}

	bucket := *opts.Bucket
	if files, ok := a.files.Load(bucket); ok {
		files.Store(*opts.Key, data)
	} else {
		return errors.New(s3.ErrCodeNoSuchBucket)
	}

	return nil
}

func (a *MockInstance) DownloadFile(ctx context.Context, output io.WriterAt, opts *s3.GetObjectInput) error {
	if opts.Bucket == nil {
		return errors.New(s3.ErrCodeNoSuchBucket)
	}
	if opts.Key == nil {
		return errors.New(s3.ErrCodeNoSuchKey)
	}

	bucket := *opts.Bucket
	if files, ok := a.files.Load(bucket); ok {
		if data, ok := files.Load(*opts.Key); ok {
			_, err := output.WriteAt(data, 0)
			return err
		} else {
			return errors.New(s3.ErrCodeNoSuchKey)
		}
	} else {
		return errors.New(s3.ErrCodeNoSuchBucket)
	}
}
