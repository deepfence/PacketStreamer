package s3

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/deepfence/PacketStreamer/pkg/config"
	"log"
	"time"
)

const (
	MaxParts = 10_000
)

var (
	header = []byte{0xde, 0xef, 0xec, 0xe0}
)

type Plugin struct {
	S3Client        *s3.Client
	Region          string
	Bucket          string
	TotalFileSize   uint64
	UploadChunkSize uint64
	UploadTimeout   time.Duration
	CannedACL       string
}

type MultipartUpload struct {
	Upload        *s3.CreateMultipartUploadOutput
	Parts         []types.CompletedPart
	Buffer        []byte
	TotalDataSent int
}

func NewPlugin(ctx context.Context, config *config.S3PluginConfig) (*Plugin, error) {
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(config.Region))

	if err != nil {
		return nil, fmt.Errorf("error loading AWS config when creating S3 client, %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	if err != nil {
		return nil, fmt.Errorf("invalid upload timeout duration, %v", err)
	}

	return &Plugin{
		S3Client:        s3Client,
		Region:          config.Region,
		Bucket:          config.Bucket,
		TotalFileSize:   uint64(*config.TotalFileSize),
		UploadChunkSize: uint64(*config.UploadChunkSize),
		UploadTimeout:   config.UploadTimeout,
		CannedACL:       config.CannedACL,
	}, nil
}

func newMultipartUpload(createOutput *s3.CreateMultipartUploadOutput) *MultipartUpload {
	return &MultipartUpload{
		Upload:        createOutput,
		Parts:         make([]types.CompletedPart, 0),
		Buffer:        make([]byte, 0),
		TotalDataSent: 0,
	}
}

func (mpu *MultipartUpload) appendToBuffer(data []byte) {
	mpu.Buffer = append(mpu.Buffer, data...)
}

//Start returns a write-only channel to which packet chunks should be written should they wish to be streamed to S3.
//It is the responsibility of the caller to close the returned channel.
func (p *Plugin) Start(ctx context.Context) chan<- string {
	inputChan := make(chan string)
	go func() {
		payloadMarker := []byte{0x0, 0x0, 0x0, 0x0}
		var mpu *MultipartUpload

		for {
			select {
			case chunk := <-inputChan:
				if mpu == nil {
					var err error
					mpu, err = p.createMultipartUpload(ctx)

					if err != nil {
						log.Printf("error creating multipart upload, stopping... - %v\n", err)
						return
					}

					mpu.appendToBuffer(header)

					if err != nil {
						log.Printf("error adding header to buffer, stopping... - %v\n", err)
						return
					}
				}
				data := []byte(chunk)
				dataLen := len(data)
				binary.LittleEndian.PutUint32(payloadMarker[:], uint32(dataLen))
				mpu.appendToBuffer(payloadMarker)
				mpu.appendToBuffer(data)

				if uint64(len(mpu.Buffer)) >= p.UploadChunkSize {
					p.flushData(ctx, mpu)
				}

				if len(mpu.Parts) == MaxParts {
					err := p.completeUpload(ctx, mpu)

					if err != nil {
						log.Printf("error completing multipart upload, stopping... - %v\n", err)
						return
					}

					mpu, err = p.createMultipartUpload(ctx)

					if err != nil {
						log.Printf("error creating multipart upload, stopping... - %v\n", err)
						return
					}

					continue
				}

				if uint64(mpu.TotalDataSent) >= p.TotalFileSize {
					err := p.completeUpload(ctx, mpu)

					if err != nil {
						log.Printf("error completing multipart upload, stopping... - %v\n", err)
						return
					}

					mpu, err = p.createMultipartUpload(ctx)

					if err != nil {
						log.Printf("error creating multipart upload, stopping... - %v\n", err)
						return
					}
				}
			case <-time.After(p.UploadTimeout):
				// write whatever data we have to
				if mpu != nil && (len(mpu.Buffer) > 0 || len(mpu.Parts) > 0) {
					log.Println("timeout internal expired - flushing...")
					p.completeUpload(ctx, mpu)
				}

				var err error
				mpu, err = p.createMultipartUpload(ctx)

				if err != nil {
					log.Printf("error creating multipart upload, stopping... - %v\n", err)
					return
				}
			case <-ctx.Done():
				p.flushData(ctx, mpu)
				return
			}
		}
	}()
	return inputChan
}

func (p *Plugin) flushData(ctx context.Context, mpu *MultipartUpload) error {
	if len(mpu.Buffer) == 0 {
		return nil
	}

	upr, err := p.S3Client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:        mpu.Upload.Bucket,
		Key:           mpu.Upload.Key,
		PartNumber:    int32(len(mpu.Parts) + 1),
		UploadId:      mpu.Upload.UploadId,
		Body:          bytes.NewBuffer(mpu.Buffer),
		ContentLength: int64(len(mpu.Buffer)),
	})

	if err != nil {
		return fmt.Errorf("error uploading part [%d] - %v", len(mpu.Parts)+1, err)
	}

	mpu.Parts = append(mpu.Parts, types.CompletedPart{
		ETag:       upr.ETag,
		PartNumber: int32(len(mpu.Parts) + 1),
	})
	mpu.TotalDataSent += len(mpu.Buffer)
	mpu.Buffer = make([]byte, 0)

	return nil
}

func (p *Plugin) completeUpload(ctx context.Context, mpu *MultipartUpload) error {
	err := p.flushData(ctx, mpu)

	if err != nil {
		return fmt.Errorf("error flushing data before upload complete, %v", err)
	}

	_, err = p.S3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   mpu.Upload.Bucket,
		Key:      mpu.Upload.Key,
		UploadId: mpu.Upload.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: mpu.Parts,
		},
	})

	if err != nil {
		return fmt.Errorf("error completing multipart upload, %v", err)
	}

	return nil
}

func (p *Plugin) createMultipartUpload(ctx context.Context) (*MultipartUpload, error) {
	t := time.Now()
	output, err := p.S3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(p.Bucket),
		//TODO: make this configurable / as intended
		Key: aws.String(fmt.Sprintf("%d-%d-%d-%d-%d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Second())),
		ACL: types.ObjectCannedACL(p.CannedACL),
	})

	if err != nil {
		return nil, fmt.Errorf("error creating multipart upload, %v", err)
	}

	return newMultipartUpload(output), nil
}
