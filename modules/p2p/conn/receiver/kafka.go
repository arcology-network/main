package receiver

import (
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
)

type KafkaReceiver struct {
	brokers       []string
	topics        []string
	group         string
	consumer      *cluster.Consumer
	onMsgReceived func(string, []byte)
}

func NewKafkaReceiver(brokers, topics []string, group string, onMsgReceived func(string, []byte)) *KafkaReceiver {
	return &KafkaReceiver{
		brokers:       brokers,
		topics:        topics,
		group:         group,
		onMsgReceived: onMsgReceived,
	}
}

func (kr *KafkaReceiver) Start() {
	config := cluster.NewConfig()
	config.Version = sarama.V2_0_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	config.Consumer.Offsets.CommitInterval = time.Second

	consumer, err := cluster.NewConsumer(kr.brokers, kr.group, kr.topics, config)
	if err != nil {
		panic(err)
	}
	kr.consumer = consumer

	go func() {
		for err := range consumer.Errors() {
			fmt.Printf("Kafka error: %v\n", err)
		}
	}()

	go func() {
		for n := range consumer.Notifications() {
			fmt.Printf("Kafka notification: %v\n", n)
		}
	}()

	go func() {
		for m := range consumer.Messages() {
			kr.onMsgReceived(m.Topic, m.Value)
		}
	}()
}

func (kr *KafkaReceiver) Stop() {
	kr.consumer.Close()
}
