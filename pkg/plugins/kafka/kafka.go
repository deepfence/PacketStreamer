package kafka

import (
	"context"
	"log"
	"time"

	"github.com/deepfence/PacketStreamer/pkg/config"
	"github.com/deepfence/PacketStreamer/pkg/file"
	"github.com/google/uuid"
	kafka "github.com/segmentio/kafka-go"
)

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type File struct {
	Id     string
	Buffer []byte
	Sent   uint64
}

func (f *File) newBuffer(size int) {
	f.Buffer = make([]byte, 0, size)
}

type IdGenerator interface {
	Generate() string
}

type FileIdGenerator struct{}

func (g *FileIdGenerator) Generate() string {
	return uuid.New().String()
}

type Plugin struct {
	Writer      KafkaWriter
	IdGenerator IdGenerator
	Topic       string
	MessageSize int
	CloseChan   chan bool
	FileSize    uint64
	CurrentFile *File
}

func NewPlugin(config *config.KafkaPluginConfig) (*Plugin, error) {
	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
		TLS:       nil,
	}
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  config.Brokers,
		Topic:    config.Topic,
		Balancer: &kafka.Hash{},
		Dialer:   dialer,
	})

	return &Plugin{
		Writer:      writer,
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

//Start produces Kafka messages containing data that is written to the returned channel
func (p *Plugin) Start(ctx context.Context) chan<- string {
	inputChan := make(chan string)
	go func() {
		defer p.Writer.Close()
		p.newFile(p.IdGenerator.Generate(), p.MessageSize)

		for {
			select {
			case pkt, more := <-inputChan:
				if !more {
					p.cleanup()
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
							//TODO: handle this better
							log.Println(err)
							return
						}

						if p.CurrentFile.Sent >= p.FileSize {
							p.newFile(p.IdGenerator.Generate(), p.MessageSize)
						} else {
							p.CurrentFile.newBuffer(p.MessageSize)
						}
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
	// we only need to clean up if there's actually data to send
	if len(p.CurrentFile.Buffer) > len(file.Header) {
		err := p.flush()
		if err != nil {
			//TODO: handle this better
			log.Println(err)
		}
	}

	close(p.CloseChan)
}

func (p *Plugin) flush() error {
	err := p.Writer.WriteMessages(context.Background(), kafka.Message{
		Topic: p.Topic,
		Key:   []byte(p.CurrentFile.Id),
		Value: p.CurrentFile.Buffer,
	})

	p.CurrentFile.Sent += uint64(len(p.CurrentFile.Buffer))

	return err
}
