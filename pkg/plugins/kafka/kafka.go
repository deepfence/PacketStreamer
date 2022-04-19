package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/deepfence/PacketStreamer/pkg/config"
)

type Plugin struct {
	Producer    *kafka.Producer
	Topic       string
	MessageSize int64
}

func NewPlugin(config *config.KafkaPluginConfig) (*Plugin, error) {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.server": config.Brokers,
		"client.id":        config.ClientId,
		"acks":             "all",
	})

	if err != nil {
		return nil, fmt.Errorf("error creating kafka producer, %w", err)
	}

	return &Plugin{
		Producer:    producer,
		Topic:       config.Topic,
		MessageSize: int64(*config.MessageSize),
	}, nil
}

func (p *Plugin) Start(ctx context.Context) chan<- string {
	inputChan := make(chan string)
	go func() {

	}()
	return inputChan
}
