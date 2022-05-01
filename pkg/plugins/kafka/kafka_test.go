package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/deepfence/PacketStreamer/pkg/file"
	"reflect"
	"testing"
)

type mockKafkaProducer struct {
	Messages []string
}

func (mkp *mockKafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	mkp.Messages = append(mkp.Messages, string(msg.Value))
	return nil
}

func (mkp *mockKafkaProducer) Close() {
	return
}

func TestPluginStart(t *testing.T) {
	tests := []struct {
		TestName         string
		Topic            string
		MessageSize      int
		ToSend           []string
		ExpectedMessages []string
	}{
		{
			TestName:         "message longer than messageSize",
			Topic:            "test",
			MessageSize:      8,
			ToSend:           []string{"regular message"},
			ExpectedMessages: []string{fmt.Sprintf("%sregu", file.Header), "lar mess", "age"},
		},
		{
			TestName:         "message shorter than messageSize",
			Topic:            "test",
			MessageSize:      100,
			ToSend:           []string{"This is a message that's not long enough"},
			ExpectedMessages: []string{fmt.Sprintf("%sThis is a message that's not long enough", file.Header)},
		},
		{
			TestName:         "short message followed by a long message",
			Topic:            "test",
			MessageSize:      20,
			ToSend:           []string{"Hello", ", the second part of this message is longer"},
			ExpectedMessages: []string{fmt.Sprintf("%sHello, the secon", file.Header), "d part of this messa", "ge is longer"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.TestName, func(t *testing.T) {
			mockProducer := &mockKafkaProducer{
				Messages: make([]string, 0),
			}

			plugin := &Plugin{
				Producer:    mockProducer,
				Topic:       tt.Topic,
				MessageSize: tt.MessageSize,
				FileSize:    getFileSizeFromMessages(t, tt.ToSend),
				CloseChan:   make(chan bool),
			}

			p := plugin.Start(context.TODO())
			{
				for _, s := range tt.ToSend {
					p.Input <- s
				}
			}
			close(p.Input)

			<-plugin.CloseChan

			if !reflect.DeepEqual(mockProducer.Messages, tt.ExpectedMessages) {
				t.Errorf("expected %v, got %v", tt.ExpectedMessages, mockProducer.Messages)
			}
		})
	}
}

func getFileSizeFromMessages(t *testing.T, sentMessages []string) uint64 {
	t.Helper()
	var fileSize = uint64(len(file.Header))

	for _, m := range sentMessages {
		fileSize += uint64(len(m))
	}

	return fileSize
}
