package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/deepfence/PacketStreamer/pkg/config"
	"log"
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
	Buffer      []byte
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
		Buffer:      make([]byte, 0, int(*config.MessageSize)),
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

				if len(p.Buffer)+len(pkt) < p.MessageSize {
					p.Buffer = append(p.Buffer, pkt...)
				} else {
					readFrom := 0
					for readFrom < len(pkt) {
						toTake := p.MessageSize - len(p.Buffer)
						if readFrom+toTake > len(pkt) {
							p.Buffer = append(p.Buffer, pkt[readFrom:]...)
							readFrom = len(pkt)

						} else {
							p.Buffer = append(p.Buffer, pkt[readFrom:readFrom+toTake]...)
							readFrom += toTake
						}

						err := p.flush()

						if err != nil {
							//TODO: handle this better
							log.Println(err)
							return
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
	if len(p.Buffer) > 4 {
		err := p.flush()
		if err != nil {
			//TODO: handle this better
			log.Println(err)
		}
	}

	close(p.CloseChan)
}

func (p *Plugin) newBuffer() {
	p.Buffer = make([]byte, 0, p.MessageSize)
	//b = append(b, header...)
}

func (p *Plugin) flush() error {
	err := p.Producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.Topic, Partition: kafka.PartitionAny},
		Value:          p.Buffer,
		Key:            []byte("packetstreamer"),
	}, nil)

	return err
}
