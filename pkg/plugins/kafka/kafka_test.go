package kafka

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/deepfence/PacketStreamer/pkg/file"
	kafka "github.com/segmentio/kafka-go"
)

type mockKafkaWriter struct {
	Messages []kafka.Message
}

func (mkp *mockKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	mkp.Messages = append(mkp.Messages, msgs...)
	return nil
}

func (mkp *mockKafkaWriter) Close() error {
	return nil
}

type mockIdGenerator struct{}

func (g *mockIdGenerator) Generate() string {
	return "test"
}

func TestPluginStart(t *testing.T) {
	tests := []struct {
		TestName         string
		Topic            string
		MessageSize      int
		ToSend           []string
		ExpectedMessages []kafka.Message
	}{
		{
			TestName:    "message longer than messageSize",
			Topic:       "test",
			MessageSize: 8,
			ToSend:      []string{"regular message"},
			ExpectedMessages: []kafka.Message{
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte(fmt.Sprintf("%sregu", file.Header)),
				},
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte("lar mess"),
				},
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte("age"),
				},
			},
		},
		{
			TestName:    "message shorter than messageSize",
			Topic:       "test",
			MessageSize: 100,
			ToSend:      []string{"This is a message that's not long enough"},
			ExpectedMessages: []kafka.Message{
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte(fmt.Sprintf("%sThis is a message that's not long enough", file.Header)),
				},
			},
		},
		{
			TestName:    "short message followed by a long message",
			Topic:       "test",
			MessageSize: 20,
			ToSend:      []string{"Hello", ", the second part of this message is longer"},
			ExpectedMessages: []kafka.Message{
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte(fmt.Sprintf("%sHello, the secon", file.Header)),
				},
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte("d part of this messa"),
				},
				{
					Topic:     "test",
					Partition: 0,
					Offset:    0,
					Key:       []byte("test"),
					Value:     []byte("ge is longer"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.TestName, func(t *testing.T) {
			mockWriter := &mockKafkaWriter{
				Messages: make([]kafka.Message, 0),
			}

			plugin := &Plugin{
				Writer:      mockWriter,
				IdGenerator: &mockIdGenerator{},
				Topic:       tt.Topic,
				MessageSize: tt.MessageSize,
				FileSize:    getFileSizeFromMessages(t, tt.ToSend),
				CloseChan:   make(chan bool),
			}

			inputChan := plugin.Start(context.TODO())
			{
				for _, s := range tt.ToSend {
					inputChan <- s
				}
			}
			close(inputChan)

			<-plugin.CloseChan

			if !reflect.DeepEqual(mockWriter.Messages, tt.ExpectedMessages) {
				t.Errorf("expected %v, got %v", tt.ExpectedMessages, mockWriter.Messages)
			}
		})
	}
}

func getFileSizeFromMessages(t *testing.T, sentMessages []string) uint64 {
	t.Helper()
	var fileSize uint64 = uint64(len(file.Header))

	for _, m := range sentMessages {
		fileSize += uint64(len(m))
	}

	return fileSize
}
