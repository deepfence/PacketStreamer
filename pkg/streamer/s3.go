package streamer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/itchyny/timefmt-go"

	"github.com/deepfence/PacketStreamer/pkg/config"
)

type S3Writer struct {
	client *s3.Client

	bucket    string
	keyFormat string
}

// S3PutObjectAPI defines the S3 API for uploading objects.
type S3PutObjectAPI interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput,
		opts ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// putObject uploads the object using the given S3 client.
func putObject(ctx context.Context, client S3PutObjectAPI,
	input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	return client.PutObject(ctx, input)
}

// newS3Writer returns a new S3 writer which does the immediate upload to S3.
// Therefore it's not supposed to be used directly, but rather with a bufio
// writer.
func newS3Writer(config *config.Config) (*S3Writer, error) {
	awsConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("could not load the AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsConfig)

	return &S3Writer{
		client:    client,
		bucket:    config.Output.S3.Bucket,
		keyFormat: config.Output.S3.KeyFormat,
	}, nil
}

// Write uploads the given packet buffer to S3.
func (w S3Writer) Write(pktData []byte) (n int, err error) {
	now := time.Now()
	key := timefmt.Format(now, w.keyFormat)
	if _, err := putObject(context.TODO(), w.client, &s3.PutObjectInput{
		Bucket: &w.bucket,
		Key:    &key,
		Body:   bytes.NewReader(pktData),
	}); err != nil {
		return 0, fmt.Errorf("could not write to S3: %w", err)
	}
	return len(pktData), nil
}

// NewS3Writer returns a new buffered S3 writer which gathers data until the
// TotalFileSize is reached and then does the actual upload to S3.
func NewS3BufWriter(config *config.Config) (*bufio.Writer, error) {
	writer, err := newS3Writer(config)
	if err != nil {
		return nil, fmt.Errorf("could not create a new S3 writer: %w", err)
	}

	return bufio.NewWriterSize(writer, int(*config.Output.S3.TotalFileSize)), nil
}
