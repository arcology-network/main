/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
