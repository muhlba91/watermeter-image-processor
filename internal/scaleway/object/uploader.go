package object

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"

	"github.com/muhlba91/watermeter-image-processor/cmd/configuration"
)

// maxHealthCheckKeys defines the maximum number of keys to retrieve when performing a health check on the S3 client.
const maxHealthCheckKeys = 1

// healthCheckTimeout defines the timeout duration for health checks related to the S3 client.
const healthCheckTimeout = 2 * time.Second

// Uploader is the struct that implements the ObjectUploader interface for Scaleway.
type Uploader struct {
	client *s3.Client
	bucket *string
	path   string
}

// NewUploader creates a new instance of the Uploader.
// configuration: The configuration data for the uploader, which may include credentials and other settings.
func NewUploader(configuration *configuration.Data) (*Uploader, error) {
	s3client := buildS3Client(configuration)

	return &Uploader{
		client: s3client,
		bucket: configuration.SCWBucket,
		path:   fmt.Sprintf(configuration.SCWBucketPath, configuration.MeterID),
	}, nil
}

// HealthCheck verifies the overall health of the S3 client by attempting to list objects in the configured bucket.
func (u *Uploader) HealthCheck() bool {
	if u.client == nil {
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer cancel()

	maxKeys := int32(maxHealthCheckKeys)
	_, err := u.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  u.bucket,
		MaxKeys: &maxKeys,
	})
	return err == nil
}

// Upload uploads the given data to Scaleway's S3-compatible object storage service.
// ctx: The context for the operation.
// data: The byte slice representing the data to be uploaded.
func (u *Uploader) Upload(ctx context.Context, data []byte) {
	if u.client == nil {
		logrus.Error("scaleway upload is not configured")
		return
	}

	key := fmt.Sprintf("%s/%s.jpg", u.path, time.Now().Format("2006/01/02/15-04"))

	logrus.Debugf("uploading object to scaleway: bucket=%s, key=%s, size=%d bytes", *u.bucket, key, len(data))
	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: u.bucket,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		logrus.Errorf("failed to upload object to scaleway: %v", err)
		return
	}
	logrus.Infof("uploaded object to scaleway: %s", key)
}

// buildS3Client builds and returns an S3 client configured for Scaleway using the provided configuration data.
// configuration: The configuration data for the uploader, which may include credentials and other settings.
func buildS3Client(configuration *configuration.Data) *s3.Client {
	if configuration.SCWRegion == nil || configuration.SCWAccessKey == nil || configuration.SCWSecretKey == nil ||
		configuration.SCWBucket == nil {
		logrus.Error(
			"scaleway upload disabled: missing SCW_REGION, SCW_ACCESS_KEY, SCW_SECRET_KEY, or SCW_BUCKET environment variables",
		)
		return nil
	}

	endpoint := fmt.Sprintf("https://s3.%s.scw.cloud", *configuration.SCWRegion)
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(*configuration.SCWRegion),
		config.WithBaseEndpoint(endpoint),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			*configuration.SCWAccessKey,
			*configuration.SCWSecretKey,
			"",
		)),
	)
	if err != nil {
		logrus.Fatalf("unable to load config for scaleway: %v", err)
	}

	return s3.NewFromConfig(cfg)
}
