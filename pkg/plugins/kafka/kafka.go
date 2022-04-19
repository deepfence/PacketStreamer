package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/deepfence/PacketStreamer/pkg/config"
)

var (
	header = []byte{0xde, 0xef, 0xec, 0xe0}
)

type Plugin struct {
	Producer    *kafka.Producer
	Topic       string
	MessageSize int
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
	}, nil
}

func (p *Plugin) Start(ctx context.Context) chan<- string {
	inputChan := make(chan string)
	go func() {
		defer p.Producer.Close()
		buffer := p.newBuffer()

		for {
			select {
			case pkt := <-inputChan:
				if len(buffer)+len(pkt) >= p.MessageSize {
					err := p.Producer.Produce(&kafka.Message{
						TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
						Value:          buffer,
					}, nil)

					if err != nil {

					}

					buffer = p.newBuffer()
				} else {
					buffer = append(buffer, pkt...)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return inputChan
}

func (p *Plugin) newBuffer() []byte {
	b := make([]byte, 0, p.MessageSize)
	b = append(b, header...)
	return b
}
