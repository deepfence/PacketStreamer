package kafka

import (
	"context"
	"fmt"
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
			TestName:         "when a message is longer than messageSize, it is split up",
			Topic:            "test",
			MessageSize:      7,
			ToSend:           []string{"regular message"},
			ExpectedMessages: []string{"regular", " message"},
		},
		{
			TestName:         "when a message is shorter than messageSize, it is sent when the channel is closed",
			Topic:            "test",
			MessageSize:      100,
			ToSend:           []string{"This is a message that's not long enough"},
			ExpectedMessages: []string{"This is a message that's not long enough"},
		},
	}

	mockProducer := &mockKafkaProducer{
		Messages: make([]string, 0),
	}

	for _, tt := range tests {
		t.Run(tt.TestName, func(t *testing.T) {
			plugin := &Plugin{
				Producer:    mockProducer,
				Topic:       tt.Topic,
				MessageSize: tt.MessageSize,
				CloseChan:   make(chan bool),
			}

			inputChan := plugin.Start(context.TODO())
			{
				for _, s := range tt.ToSend {
					fmt.Printf("sending %s", s)
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