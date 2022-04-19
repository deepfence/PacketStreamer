package kafka

import "github.com/confluentinc/confluent-kafka-go/kafka"

type Plugin struct {
	Producer kafka.Producer
}

func NewPlugin() *Plugin {
	return &Plugin{}
}
