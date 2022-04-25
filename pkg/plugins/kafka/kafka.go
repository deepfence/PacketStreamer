package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/file"
	pluginTypes "github.com/deepfence/PacketStreamer/pkg/plugins/types"
	"github.com/google/uuid"
)

type KafkaProducer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

type File struct {
	Id     string
	Buffer []byte
	Sent   uint64
}

func (f *File) newBuffer(size int) {
	f.Buffer = make([]byte, 0, size)
}

type Plugin struct {
	Producer    KafkaProducer
	Topic       string
	MessageSize int
	CloseChan   chan bool
	FileSize    uint64
	CurrentFile *File
}

func NewPlugin(config *config.KafkaPluginConfig) (*Plugin, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": config.Brokers,
		"client.id":         config.ClientId,
		"acks":              config.Acks,
	})

	if err != nil {
		return nil, fmt.Errorf("error creating kafka producer, %w", err)
	}

	return &Plugin{
		Producer:    producer,
		Topic:       config.Topic,
		MessageSize: int(*config.MessageSize),
		FileSize:    uint64(*config.FileSize),
		CloseChan:   make(chan bool),
	}, nil
}

func (p *Plugin) newFile(id string, messageSize int) {
	p.CurrentFile = &File{
		Id:     id,
		Buffer: make([]byte, 0, messageSize),
	}

	p.CurrentFile.Buffer = append(p.CurrentFile.Buffer, file.Header...)
}

//Start returns a struct which contains a write-only channel to which packet chunks should be written should they wish to be streamed to Kafka.
//Errors produced by this method will be sent on the contained Error channel.
//It is the responsibility of the caller to close the returned Input channel.
//This method will handle closure of the returned Error channel.
func (p *Plugin) Start(ctx context.Context) pluginTypes.RunningPlugin {
	inputChan := make(chan string)
	errorChan := make(chan error)
	go func() {
		defer func() {
			p.Producer.Close()
			close(errorChan)
		}()

		p.newFile(generateFileId(), p.MessageSize)

		for {
			select {
			case pkt, more := <-inputChan:
				if !more {
					p.cleanup(errorChan)
					return
				}

				if len(p.CurrentFile.Buffer)+len(pkt) < p.MessageSize {
					p.CurrentFile.Buffer = append(p.CurrentFile.Buffer, pkt...)
				} else {
					// chunk the message so that it fits in our configured message size
					readFrom := 0
					for readFrom < len(pkt) {
						toTake := p.MessageSize - len(p.CurrentFile.Buffer)
						if readFrom+toTake > len(pkt) {
							p.CurrentFile.Buffer = append(p.CurrentFile.Buffer, pkt[readFrom:]...)
							readFrom = len(pkt)

						} else {
							p.CurrentFile.Buffer = append(p.CurrentFile.Buffer, pkt[readFrom:readFrom+toTake]...)
							readFrom += toTake
						}

						err := p.flush()

						if err != nil {
							errorChan <- err
							return
						}

						if p.CurrentFile.Sent >= p.FileSize {
							p.newFile(generateFileId(), p.MessageSize)
						} else {
							p.CurrentFile.newBuffer(p.MessageSize)
						}
					}
				}
			case <-ctx.Done():
				p.cleanup(errorChan)
				return
			}
		}
	}()
	return pluginTypes.RunningPlugin{
		Input:  inputChan,
		Errors: errorChan,
	}
}

func (p *Plugin) cleanup(errorChan chan error) {
	// we only need to clean up if there's actually data to send
	if len(p.CurrentFile.Buffer) > len(file.Header) {
		err := p.flush()
		if err != nil {
			errorChan <- err
		}
	}

	close(p.CloseChan)
}

func (p *Plugin) flush() error {
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          p.CurrentFile.Buffer,
		Key:            []byte(p.CurrentFile.Id),
	}, nil)

	p.CurrentFile.Sent += uint64(len(p.CurrentFile.Buffer))

	return err
}

func generateFileId() string {
	return uuid.New().String()
}
