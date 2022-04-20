package kafka

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/kafka"
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
			ExpectedMessages: []string{"regular ", "message"},
		},
		{
			TestName:         "message shorter than messageSize",
			Topic:            "test",
			MessageSize:      100,
			ToSend:           []string{"This is a message that's not long enough"},
			ExpectedMessages: []string{"This is a message that's not long enough"},
		},
		{
			TestName:         "short message followed by a long message",
			Topic:            "test",
			MessageSize:      20,
			ToSend:           []string{"Hello", ", the second part of this message is longer"},
			ExpectedMessages: []string{"Hello, the second pa", "rt of this message i", "s longer"},
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
				CloseChan:   make(chan bool),
				Buffer:      make([]byte, 0, tt.MessageSize),
			}

			inputChan := plugin.Start(context.TODO())
			{
				for _, s := range tt.ToSend {
					inputChan <- s
				}
			}
			close(inputChan)

			<-plugin.CloseChan

			if !reflect.DeepEqual(mockProducer.Messages, tt.ExpectedMessages) {
				t.Errorf("expected %v, got %v", tt.ExpectedMessages, mockProducer.Messages)
			}
		})
	}
}
