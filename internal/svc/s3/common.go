package s3

import "github.com/aws/aws-sdk-go/aws"

var (
	DefaultCacheControl = aws.String("public, max-age=15552000")
	AclPublicRead       = aws.String("public-read")
	AclPrivate          = aws.String("private")
)

type Options struct {
	Region      string
	Endpoint    string
	AccessToken string
	SecretKey   string
}
