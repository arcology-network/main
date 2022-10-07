package config

import (
	"fmt"
	"testing"

	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/mock/kafka"
	"github.com/HPISTechnologies/component-lib/mock/rpc"
	"github.com/HPISTechnologies/component-lib/streamer"
	"github.com/spf13/viper"
)

func TestPrintWorkers(t *testing.T) {
	workers := make(map[string]actor.IWorkerEx)
	for name, creator := range actor.Factory.Registry() {
		workers[name] = creator(1, "unittester")
	}
	PrintWorkers(workers)
}

// func TestPrintAllWorkers(t *testing.T) {
// 	loadConfig(t, "./global.json", "./kafka.json", "./allapp.json")
// }

// func TestPrintAllWorkersWithoutKafka(t *testing.T) {
// 	loadConfig(t, "./global.json", "./kafka-empty.json", "./allapp.json")
// }

func TestLoadAppConfig(t *testing.T) {
	config := LoadAppConfig("../modules/exec/exec.json")
	t.Log(config)
}

func TestLoadExecSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/exec/exec.json")
}

func TestLoadArbitratorSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/arbitrator/arbitrator.json")
}

func TestLoadTppSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/tpp/tpp.json")
}

func TestLoadPoolSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/pool/pool.json")
}

func TestLoadCoreSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/core/core.json")
}

func TestLoadSchedulingSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/scheduler/scheduler.json")
}

func TestLoadConsensusSvcConfig(t *testing.T) {
	viper.Set("home", "./tmroot")
	loadConfig(t, "./global.json", "./kafka.json", "../modules/consensus/consensus.json")
}

func TestGatewaySvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/gateway/gateway.json")
}

func TestEthApiSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/eth-api/eth-api.json")
}

func TestLoadNewStorageSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/storage/storage.json")
	// writeArch("../modules/storage/storage.json", "storage.dot")
}

func TestLoadNewReceiptHashingSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/receipt-hashing/receipt-hashing.json")
	// writeArch("../modules/receipt-hashing/receipt-hashing.json", "receipt-hashing.dot")
}

func TestP2pGatewaySvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/p2p/p2p-gateway.json")
}

func TestP2pConnSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/p2p/p2p-conn.json")
}

func TestStateSyncSvcConfig(t *testing.T) {
	loadConfig(t, "./global.json", "./kafka.json", "../modules/state-sync/state-sync.json")
}

func loadConfig(t *testing.T, globalConfigFile, kafkaConfigFile, appConfigFile string) {
	DownloaderCreator = kafka.NewDownloaderCreator(t)
	UploaderCreator = kafka.NewUploaderCreator(t)
	intf.RPCCreator = rpc.NewRPCServerInitializer(t)

	globalConfig := LoadGlobalConfig(globalConfigFile)
	appConfig := LoadAppConfig(appConfigFile)
	broker := streamer.NewStatefulStreamer()
	workers := appConfig.InitApp(broker, globalConfig)
	t.Log(workers)
	PrintWorkers(workers)

	var inputs []string
	outputs := make(map[string]int)
	for _, w := range workers {
		in, _ := w.Inputs()
		inputs = actor.MergeInputs(inputs, in)
		outputs = actor.MergeOutputs(outputs, w.Outputs())
	}
	t.Log(inputs)
	t.Log(outputs)

	kafkaConfig := LoadKafkaConfig(kafkaConfigFile)
	// GenerateDot(workers, kafkaConfig, "./arch.dot")
	downloaders, uploaders := kafkaConfig.InitKafka(broker, workers, globalConfig, appConfig)
	t.Log(downloaders)
	t.Log(uploaders)

	var msgs []*actor.Message
	for _, worker := range workers {
		if _, ok := worker.(actor.Initializer); ok {
			msgs = append(msgs, worker.(actor.Initializer).InitMsgs()...)
		}
	}
	t.Log(msgs)
}

func TestGenerateArch(t *testing.T) {
	DownloaderCreator = kafka.NewDownloaderCreator(t)
	UploaderCreator = kafka.NewUploaderCreator(t)
	intf.RPCCreator = rpc.NewRPCServerInitializer(t)
	viper.Set("home", "./tmroot")

	services := make(map[string]map[string]actor.IWorkerEx)
	services["exec"] = writeArch("../modules/exec/exec.json", "exec.dot")
	// services["eshing"] = writeArch("eshing.json", "eshing.dot")
	services["core"] = writeArch("../modules/core/core.json", "core.dot")
	services["gateway"] = writeArch("../modules/gateway/gateway.json", "gateway.dot")
	services["consensus"] = writeArch("../modules/consensus/consensus.json", "consensus.dot")
	services["eth-api"] = writeArch("../modules/eth-api/eth-api.json", "eth-api.dot")
	// services["generic-hashing"] = writeArch("generic-hashing.json", "generic-hashing.dot")
	services["receipt-hashing"] = writeArch("../modules/receipt-hashing/receipt-hashing.json", "receipt-hashing.dot")
	services["pool"] = writeArch("../modules/pool/pool.json", "pool.dot")
	services["scheduler"] = writeArch("../modules/scheduler/scheduler.json", "scheduler.dot")
	// services["storage"] = writeArch("storage.json", "storage.dot")
	services["storage"] = writeArch("../modules/storage/storage.json", "storage.dot")
	services["tpp"] = writeArch("../modules/tpp/tpp.json", "tpp.dot")
	services["arbitrator"] = writeArch("../modules/arbitrator/arbitrator.json", "arbitrator.dot")
	services["coordinator"] = writeArch("../modules/coordinator/coordinator.json", "coordinator.dot")
	services["p2p-conn"] = writeArch("../modules/p2p/p2p-conn.json", "p2p-conn.dot")
	services["p2p-gateway"] = writeArch("../modules/p2p/p2p-gateway.json", "p2p-gateway.dot")
	services["state-sync"] = writeArch("../modules/state-sync/state-sync.json", "state-sync.dot")

	highlights := map[string]string{
		"inclusive":               "green",
		"blockCompleted":          "red",
		"parentinfo":              "blue",
		"euResults":               "yellow",
		"blockstart":              "cyan",
		"pendingblock":            "brown",
		"external.blockstart":     "magenta",
		"external.reapcommand":    "magenta",
		"external.txblocks":       "magenta",
		"external.blockcompleted": "magenta",
		"external.reapinglist":    "magenta",
		"external.apphash":        "magenta",
	}

	kafkaConfig := LoadKafkaConfig("./kafka.json")
	var g graph
	index := 1
	port := 1
	inputPorts := make(map[string][]string)
	outputPorts := make(map[string][]string)
	for sname, workers := range services {
		var n node
		n.name = sname

		inputDict := make(map[string]struct{})
		outputDict := make(map[string]struct{})
		for _, worker := range workers {
			inputs, _ := worker.Inputs()
			for _, in := range inputs {
				_, topic := kafkaConfig.getServerTopic(in)
				if topic != "" {
					if _, ok := inputDict[in]; ok {
						continue
					} else {
						inputDict[in] = struct{}{}
					}

					if color, ok := highlights[in]; ok {
						n.inputs = append(n.inputs, fmt.Sprintf("<TD BGCOLOR=\"%s\">%s</TD>", color, in))
					} else {
						n.inputs = append(n.inputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, in))
						inputPorts[in] = append(inputPorts[in], fmt.Sprintf("thread%d:p%d", index, port))
						port++
					}
				}
			}

			outputs := worker.Outputs()
			for out := range outputs {
				_, topic := kafkaConfig.getServerTopic(out)
				if topic != "" {
					if _, ok := outputDict[out]; ok {
						continue
					} else {
						outputDict[out] = struct{}{}
					}

					if color, ok := highlights[out]; ok {
						n.outputs = append(n.outputs, fmt.Sprintf("<TD BGCOLOR=\"%s\">%s</TD>", color, out))
					} else {
						n.outputs = append(n.outputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, out))
						outputPorts[out] = append(outputPorts[out], fmt.Sprintf("thread%d:p%d", index, port))
						port++
					}
				}
			}
		}

		g.nodes = append(g.nodes, n)
		index++
	}

	for out, ports := range outputPorts {
		for _, port := range ports {
			for _, inPort := range inputPorts[out] {
				g.edges = append(g.edges, edge{
					from: port,
					to:   inPort,
				})
			}
		}
	}

	writeDot(g, "arch.dot")
}

func writeArch(app string, file string) map[string]actor.IWorkerEx {
	globalConfig := LoadGlobalConfig("./global.json")
	appConfig := LoadAppConfig(app)
	broker := streamer.NewStatefulStreamer()
	workers := appConfig.InitApp(broker, globalConfig)
	kafkaConfig := LoadKafkaConfig("./kafka.json")
	GenerateDot(workers, kafkaConfig, file)
	return workers
}
