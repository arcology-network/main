package sender

import (
	"fmt"

	"github.com/Shopify/sarama"
)

type KafkaSender struct {
	producer sarama.AsyncProducer
	topic    string
}

func NewKafkaSender(brokers []string, topic string) (*KafkaSender, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Flush.Messages = 1
	config.Producer.Return.Errors = true
	config.Version = sarama.V2_0_0_0

	producer, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	go func() {
		for err := range producer.Errors() {
			fmt.Printf("Kafka error: %v\n", err)
		}
	}()

	return &KafkaSender{
		producer: producer,
		topic:    topic,
	}, nil
}

func (ks *KafkaSender) Close() {
	ks.producer.Close()
}

func (ks *KafkaSender) Send(b []byte) {
	ks.producer.Input() <- &sarama.ProducerMessage{
		Topic: ks.topic,
		Value: sarama.ByteEncoder(b),
	}
}
