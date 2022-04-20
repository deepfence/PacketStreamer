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
	buffer      []byte
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
		buffer:      make([]byte, 0, int(*config.MessageSize)),
	}, nil
}

func (p *Plugin) Start(ctx context.Context) chan<- string {
	inputChan := make(chan string)
	go func() {
		defer p.Producer.Close()
		p.newBuffer()

		for {
			select {
			case pkt, more := <-inputChan:
				if !more {
					p.cleanup()
					return
				}

				if len(p.buffer)+len(pkt) < p.MessageSize {
					p.buffer = append(p.buffer, pkt...)
				} else {
					readFrom := 0
					for readFrom < len(pkt) {
						toTake := p.MessageSize - len(p.buffer)
						if readFrom+toTake > len(pkt) {
							p.buffer = append(p.buffer, pkt[readFrom:]...)
							readFrom = len(pkt)

						} else {
							p.buffer = append(p.buffer, pkt[readFrom:readFrom+toTake]...)
							readFrom += toTake
						}

						err := p.flush()

						if err != nil {
							//TODO: ??? - _probably_ just log this?
						}

						p.newBuffer()
					}
				}
			case <-ctx.Done():
				p.cleanup()
				return
			}
		}
	}()
	return inputChan
}

func (p *Plugin) cleanup() {
	if len(p.buffer) > 4 {
		err := p.flush()
		if err != nil {
			//TODO: ??? - _probably_ just log this?
		}
	}

	close(p.CloseChan)
}

func (p *Plugin) newBuffer() {
	p.buffer = make([]byte, 0, p.MessageSize)
	//b = append(b, header...)
}

func (p *Plugin) flush() error {
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          p.buffer,
	}, nil)

	return err
}
