package s3

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/SevenTV/Common/sync_map"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/seventv/api/internal/instance"
)

type MockInstance struct {
	files     *sync_map.Map[string, *sync_map.Map[string, []byte]]
	connected bool
}

func NewMock(ctx context.Context, files map[string]map[string][]byte) (instance.S3, error) {
	mp := &sync_map.Map[string, *sync_map.Map[string, []byte]]{}
	for k, v := range files {
		mp.Store(k, sync_map.FromStdMap(v))
	}

	return &MockInstance{
		files:     mp,
		connected: true,
	}, nil
}

func (a *MockInstance) SetConnected(connected bool) {
	a.connected = connected
}

func (a *MockInstance) ListBuckets(ctx context.Context) (*s3.ListBucketsOutput, error) {
	if !a.connected {
		return nil, http.ErrHandlerTimeout
	}

	resp := &s3.ListBucketsOutput{}

	a.files.Range(func(key string, value *sync_map.Map[string, []byte]) bool {
		resp.Buckets = append(resp.Buckets, &s3.Bucket{
			Name:         aws.String(key),
			CreationDate: aws.Time(time.Now()),
		})

		return true
	})

	return resp, nil
}

func (a *MockInstance) UploadFile(ctx context.Context, opts *s3manager.UploadInput) error {
	if !a.connected {
		return http.ErrHandlerTimeout
	}

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
	if !a.connected {
		return http.ErrHandlerTimeout
	}

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
