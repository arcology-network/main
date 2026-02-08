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
	"fmt"
	"testing"

	"github.com/spf13/viper"
)

func TestActorNode(t *testing.T) {
	appConfigFile := "../modules/pool/pool.yaml"
	appConfig, _ := LoadAppConfig(appConfigFile)

	// rawActors := appConfig.Actors

	// normalized := make(map[string]map[string]interface{})

	// for name, v := range rawActors {
	// 	normalized[name] = normalizeMap(v).(map[string]interface{})
	// }

	// actorTree := ParseActors(normalized)

	actorTree := ParseActors(appConfig.Actors)
	for name, actor := range actorTree {
		fmt.Println(name)
		fmt.Println(actor.Name)
		fmt.Println("params:", actor.Params)
		fmt.Println("subs:", len(actor.Subs))
		for _, actorSub := range actor.Subs {
			fmt.Println(actorSub.Name)
			fmt.Println("params:", actorSub.Params)
			fmt.Println("subs:", len(actorSub.Subs))
		}
	}
}

func TestLoadArbitrator(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/arbitrator/arbitrator.yaml")
}

func TestLoadConsensus(t *testing.T) {
	viper.Set("home", "./tmroot")
	LoadConf("./global.yaml", "./jet.yaml", "../modules/consensus/consensus.yaml")
}

func TestLoadCoordinator(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/coordinator/coordinator.yaml")
}

func TestLoadCore(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/core/core.yaml")
}

func TestLoadEthApi(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/eth-api/eth-api.yaml")
}

func TestLoadExec(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/exec/exec.yaml")
}

func TestLoadGateway(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/gateway/gateway.yaml")
}

func TestLoadPool(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/pool/pool.yaml")
}

func TestLoadReceiptHashing(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/receipt-hashing/receipt-hashing.yaml")
}

func TestLoadScheduler(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/scheduler/scheduler.yaml")
}

func TestLoadStorage(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/storage/storage.yaml")
}

func TestLoadTpp(t *testing.T) {
	LoadConf("./global.yaml", "./jet.yaml", "../modules/tpp/tpp.yaml")
}

// func loadConfig(t *testing.T, globalConfigFile, kafkaConfigFile, appConfigFile string) {
// 	DownloaderCreator = kafka.NewDownloaderCreator(t)
// 	UploaderCreator = kafka.NewUploaderCreator(t)
// 	intf.RPCCreator = rpc.NewRPCServerInitializer(t)

// 	globalConfig := LoadGlobalConfig(globalConfigFile)
// 	appConfig := LoadAppConfig(appConfigFile)
// 	broker := brokerpk.NewStatefulStreamer()
// 	workers := appConfig.InitApp(broker, globalConfig)
// 	t.Log(workers)
// 	PrintWorkers(workers)

// 	var inputs []string
// 	outputs := make(map[string]int)
// 	for _, w := range workers {
// 		in, _ := w.Inputs()
// 		inputs = actor.MergeInputs(inputs, in)
// 		outputs = actor.MergeOutputs(outputs, w.Outputs())
// 	}
// 	t.Log(inputs)
// 	t.Log(outputs)

// 	kafkaConfig := LoadKafkaConfig(kafkaConfigFile)
// 	// GenerateDot(workers, kafkaConfig, "./arch.dot")
// 	downloaders, uploaders := kafkaConfig.InitKafka(broker, workers, globalConfig, appConfig)
// 	t.Log(downloaders)
// 	t.Log(uploaders)

// 	var msgs []*actor.Message
// 	for _, worker := range workers {
// 		if _, ok := worker.(actor.Initializer); ok {
// 			msgs = append(msgs, worker.(actor.Initializer).InitMsgs()...)
// 		}
// 	}
// 	t.Log(msgs)
// }

// func TestGenerateArch(t *testing.T) {
// 	DownloaderCreator = kafka.NewDownloaderCreator(t)
// 	UploaderCreator = kafka.NewUploaderCreator(t)
// 	intf.RPCCreator = rpc.NewRPCServerInitializer(t)
// 	viper.Set("home", "./tmroot")

// 	services := make(map[string]map[string]actor.IWorkerEx)
// 	services["exec"] = writeArch("../modules/exec/exec.json", "exec.dot")
// 	services["core"] = writeArch("../modules/core/core.json", "core.dot")
// 	services["gateway"] = writeArch("../modules/gateway/gateway.json", "gateway.dot")
// 	services["consensus"] = writeArch("../modules/consensus/consensus.json", "consensus.dot")
// 	services["eth-api"] = writeArch("../modules/eth-api/eth-api.json", "eth-api.dot")
// 	services["receipt-hashing"] = writeArch("../modules/receipt-hashing/receipt-hashing.json", "receipt-hashing.dot")
// 	services["pool"] = writeArch("../modules/pool/pool.json", "pool.dot")
// 	services["scheduler"] = writeArch("../modules/scheduler/scheduler.json", "scheduler.dot")
// 	services["storage"] = writeArch("../modules/storage/storage.json", "storage.dot")
// 	services["tpp"] = writeArch("../modules/tpp/tpp.json", "tpp.dot")
// 	services["arbitrator"] = writeArch("../modules/arbitrator/arbitrator.json", "arbitrator.dot")
// 	services["coordinator"] = writeArch("../modules/coordinator/coordinator.json", "coordinator.dot")
// 	services["p2p-conn"] = writeArch("../modules/p2p/p2p-conn.json", "p2p-conn.dot")
// 	services["p2p-gateway"] = writeArch("../modules/p2p/p2p-gateway.json", "p2p-gateway.dot")
// 	services["state-sync"] = writeArch("../modules/state-sync/state-sync.json", "state-sync.dot")
// 	services["eth-api"] = writeArch("../modules/eth-api/eth-api.json", "eth-api.dot")
// 	services["tx-sync"] = writeArch("../modules/tx-sync/tx-sync.json", "tx-sync.dot")

// 	highlights := map[string]string{
// 		"inclusive":               "green",
// 		"blockCompleted":          "red",
// 		"parentinfo":              "blue",
// 		"euResults":               "yellow",
// 		"blockstart":              "cyan",
// 		"pendingblock":            "brown",
// 		"external.blockstart":     "magenta",
// 		"external.reapcommand":    "magenta",
// 		"external.txblocks":       "magenta",
// 		"external.blockcompleted": "magenta",
// 		"external.reapinglist":    "magenta",
// 		"external.apphash":        "magenta",
// 	}

// 	kafkaConfig := LoadKafkaConfig("./kafka.json")
// 	var g graph
// 	index := 1
// 	port := 1
// 	inputPorts := make(map[string][]string)
// 	outputPorts := make(map[string][]string)
// 	for sname, workers := range services {
// 		var n node
// 		n.name = sname

// 		inputDict := make(map[string]struct{})
// 		outputDict := make(map[string]struct{})
// 		for _, worker := range workers {
// 			inputs, _ := worker.Inputs()
// 			for _, in := range inputs {
// 				_, topic := kafkaConfig.getServerTopic(in)
// 				if topic != "" {
// 					if _, ok := inputDict[in]; ok {
// 						continue
// 					} else {
// 						inputDict[in] = struct{}{}
// 					}

// 					if color, ok := highlights[in]; ok {
// 						n.inputs = append(n.inputs, fmt.Sprintf("<TD BGCOLOR=\"%s\">%s</TD>", color, in))
// 					} else {
// 						n.inputs = append(n.inputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, in))
// 						inputPorts[in] = append(inputPorts[in], fmt.Sprintf("thread%d:p%d", index, port))
// 						port++
// 					}
// 				}
// 			}

// 			outputs := worker.Outputs()
// 			for out := range outputs {
// 				_, topic := kafkaConfig.getServerTopic(out)
// 				if topic != "" {
// 					if _, ok := outputDict[out]; ok {
// 						continue
// 					} else {
// 						outputDict[out] = struct{}{}
// 					}

// 					if color, ok := highlights[out]; ok {
// 						n.outputs = append(n.outputs, fmt.Sprintf("<TD BGCOLOR=\"%s\">%s</TD>", color, out))
// 					} else {
// 						n.outputs = append(n.outputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, out))
// 						outputPorts[out] = append(outputPorts[out], fmt.Sprintf("thread%d:p%d", index, port))
// 						port++
// 					}
// 				}
// 			}
// 		}

// 		g.nodes = append(g.nodes, n)
// 		index++
// 	}

// 	for out, ports := range outputPorts {
// 		for _, port := range ports {
// 			for _, inPort := range inputPorts[out] {
// 				g.edges = append(g.edges, edge{
// 					from: port,
// 					to:   inPort,
// 				})
// 			}
// 		}
// 	}

// 	writeDot(g, "arch.dot")
// }

// func writeArch(app string, file string) map[string]actor.IWorkerEx {
// 	globalConfig := LoadGlobalConfig("./global.json")
// 	appConfig := LoadAppConfig(app)
// 	broker := brokerpk.NewStatefulStreamer()
// 	workers := appConfig.InitApp(broker, globalConfig)
// 	kafkaConfig := LoadKafkaConfig("./kafka.json")
// 	GenerateDot(workers, kafkaConfig, file)
// 	return workers
// }
