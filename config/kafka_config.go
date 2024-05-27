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

package config

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/arcology-network/streamer/actor"
	brokerpk "github.com/arcology-network/streamer/broker"
	"github.com/arcology-network/streamer/kafka"
)

type KafkaConfig map[string]map[string][]string

type KafkaDownloaderCreator func(concurrency int, groupId string, topics, messageTypes []string, mqaddr string) actor.IWorkerEx
type KafkaUploaderCreator func(concurrency int, groupId string, messages map[string]string, mqaddr string) actor.IWorkerEx

var (
	DownloaderCreator KafkaDownloaderCreator = kafka.NewKafkaDownloader
	UploaderCreator   KafkaUploaderCreator   = kafka.NewKafkaUploader
)

func LoadKafkaConfig(file string) KafkaConfig {
	jsonFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}

	var config KafkaConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		panic(err)
	}
	return config
}

func (config KafkaConfig) InitKafka(
	broker *brokerpk.StatefulStreamer,
	workers map[string]actor.IWorkerEx,
	globalConfig GlobalConfig,
	appConfig AppConfig,
) (downloaders []actor.IWorkerEx, uploaders []actor.IWorkerEx) {
	var inputs []string
	outputs := make(map[string]int)
	for _, w := range workers {
		ins, _ := w.Inputs()
		inputs = actor.MergeInputs(inputs, ins)
		outputs = actor.MergeOutputs(outputs, w.Outputs())
	}

	for name, worker := range workers {
		needCleaner := make(map[string]struct{})
		ins, isConjunction := worker.Inputs()
		for _, input := range ins {
			server, _ := config.getServerTopic(input)
			if server == "" {
				continue
			}

			if _, ok := outputs[input]; ok {
				needCleaner[input] = struct{}{}
			}
		}

		if len(needCleaner) != 0 {
			if isConjunction {
				panic("not supported.")
			}

			var msgs []string
			for msg := range needCleaner {
				msgs = append(msgs, msg)
			}

			if _, ok := worker.(actor.LinkedActor); ok {
				workers[name] = actor.NewMsgCleaner(actor.MsgsOnlyFrom(msgs, "downloader"))
				workers[name].(actor.LinkedActor).Next(worker.(actor.LinkedActor))
			} else {
				workers[name] = actor.NewMsgCleaner(actor.MsgsOnlyFrom(msgs, "downloader"))
				workers[name].(actor.LinkedActor).EndWith(worker)
			}
		}

		workerActor := actor.NewActorEx(name, broker, workers[name])
		if isConjunction {
			workerActor.Connect(brokerpk.NewConjunctions(workerActor))
		} else {
			workerActor.Connect(brokerpk.NewDisjunctions(workerActor, 1))
		}
	}

	type downloaderParam struct {
		messages []string
		topics   []string
	}
	downloaderParams := make(map[string]downloaderParam)
	for _, input := range inputs {
		server, topic := config.getServerTopic(input)
		if server == "" {
			continue
		}

		param, ok := downloaderParams[server]
		if !ok {
			param = downloaderParam{}
		}
		param.messages = append(param.messages, input)
		param.topics = append(param.topics, topic)
		downloaderParams[server] = param
	}

	for server, param := range downloaderParams {
		downloader := DownloaderCreator(
			globalConfig.GetConcurrency(appConfig.Settings.ServiceName),
			appConfig.Settings.ServiceName,
			strSliceDedup(param.topics),
			strSliceDedup(param.messages),
			globalConfig.Kafka[server],
		)
		downloaders = append(downloaders, downloader)
		downloaderActor := actor.NewActorEx(
			"downloader",
			broker,
			downloader,
		)
		downloaderActor.Connect(brokerpk.NewDisjunctions(downloaderActor, 100))
	}

	type uploaderParam struct {
		messages map[string]string
	}
	uploaderParams := make(map[string]uploaderParam)
	for output := range outputs {
		server, topic := config.getServerTopic(output)
		if server == "" {
			continue
		}

		param, ok := uploaderParams[server]
		if !ok {
			param = uploaderParam{
				messages: make(map[string]string),
			}
		}
		param.messages[output] = topic
		uploaderParams[server] = param
	}

	for server, param := range uploaderParams {
		cleaner := actor.NewMsgCleaner(actor.NotFrom("downloader"))
		uploader := UploaderCreator(
			globalConfig.GetConcurrency(appConfig.Settings.ServiceName),
			appConfig.Settings.ServiceName,
			param.messages,
			globalConfig.Kafka[server],
		)
		cleaner.EndWith(uploader)
		uploaders = append(uploaders, uploader)
		uploaderActor := actor.NewActorEx(
			"uploader",
			broker,
			cleaner,
		)
		uploaderActor.Connect(brokerpk.NewDisjunctions(uploaderActor, 100))
	}
	return
}

func (config KafkaConfig) getServerTopic(message string) (server string, topic string) {
	for server, topics := range config {
		for topic, messages := range topics {
			for _, msg := range messages {
				if message == msg {
					return server, topic
				}
			}
		}
	}
	return "", ""
}

func strSliceDedup(slice []string) []string {
	strMap := make(map[string]struct{})
	for _, str := range slice {
		strMap[str] = struct{}{}
	}

	strs := make([]string, 0, len(strMap))
	for str := range strMap {
		strs = append(strs, str)
	}
	return strs
}
