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

type KafkaProducer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

type Plugin struct {
	Producer    KafkaProducer
	Topic       string
	MessageSize int
	CloseChan   chan bool
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
		CloseChan:   make(chan bool),
	}, nil
}

func (p *Plugin) Start(ctx context.Context) chan<- string {
	inputChan := make(chan string)
	go func() {
		defer p.Producer.Close()
		buffer := p.newBuffer()

		for {
			select {
			case pkt, more := <-inputChan:
				if !more {
					if len(buffer) > 4 {
						err := p.flush(buffer)
						if err != nil {
							//TODO: ??? - _probably_ just log this?
						}
					}

					close(p.CloseChan)

					return
				}

				if len(buffer)+len(pkt) < p.MessageSize {
					buffer = append(buffer, pkt...)
				} else {
					readFrom := 0
					for readFrom < len(pkt) {
						toTake := p.MessageSize - len(buffer)
						if readFrom+toTake > len(pkt) {
							buffer = append(buffer, pkt[readFrom:]...)
							readFrom = len(pkt)

						} else {
							buffer = append(buffer, pkt[readFrom:readFrom+toTake]...)
							readFrom += toTake
						}

						err := p.flush(buffer)

						if err != nil {
							//TODO: ??? - _probably_ just log this?
						}

						buffer = p.newBuffer()
					}
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
	//b = append(b, header...)
	return b
}

func (p *Plugin) flush(buffer []byte) error {
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          buffer,
	}, nil)

	return err
}
