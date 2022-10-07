package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/HPISTechnologies/component-lib/actor"
	_ "github.com/HPISTechnologies/component-lib/aggregator/v3"
	intf "github.com/HPISTechnologies/component-lib/interface"
	_ "github.com/HPISTechnologies/component-lib/storage"
	"github.com/HPISTechnologies/component-lib/streamer"
	_ "github.com/HPISTechnologies/main/modules"
)

type Settings struct {
	ServiceName          string
	PrometheusListenAddr string                 `json:"premetheus_listen_addr"`
	Envs                 map[string]interface{} `json:"envs"`
}

func (s Settings) Env(key string) interface{} {
	return s.Envs[key]
}

type MsgOperation struct {
	Type   string      `json:"type"`
	Params interface{} `json:"params"`
}

type Interface struct {
	Name    string                 `json:"name"`
	Service string                 `json:"service"`
	Params  map[string]interface{} `json:"params"`
}

type AppConfig struct {
	Settings   Settings                          `json:"settings"`
	Actors     map[string]map[string]interface{} `json:"actors"`
	MsgOps     []MsgOperation                    `json:"msg_ops"`
	Interfaces []Interface                       `json:"interfaces"`
	StartMsgs  []actor.Message                   `json:"start_msgs"`

	workersDict map[string]actor.IWorkerEx
}

func LoadAppConfig(file string) AppConfig {
	jsonFile, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	bytes, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		panic(err)
	}

	var config AppConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		panic(err)
	}

	config.Settings.ServiceName = strings.Split(filepath.Base(file), ".")[0]
	config.workersDict = make(map[string]actor.IWorkerEx)
	return config
}

func (config *AppConfig) InitApp(broker *streamer.StatefulStreamer, globalConfig GlobalConfig) map[string]actor.IWorkerEx {
	intf.Router.SetZkServers([]string{globalConfig.Zookeeper})
	var rpcs []string
	for rpc := range globalConfig.Rpc {
		rpcs = append(rpcs, rpc)
	}
	intf.Router.SetAvailableServices(rpcs)

	for name, params := range config.Actors {
		if _, ok := config.workersDict[name]; ok {
			continue
		}

		if name[0] == '-' {
			actors := strings.Split(name[1:], "-")
			if len(actors) < 2 {
				panic("one actor cannot form a chain")
			}

			var subWorkers []actor.IWorkerEx
			needHeightController := false
			for _, name := range actors {
				params := params[name]
				if params == nil {
					params = make(map[string]interface{})
				}
				worker := config.createWorker(name, params.(map[string]interface{}), globalConfig)
				if _, ok := worker.(actor.HeightSensitive); ok {
					needHeightController = true
				}
				subWorkers = append(subWorkers, worker)
			}

			var baseWorker actor.LinkedActor
			var previous actor.LinkedActor
			if needHeightController {
				baseWorker = actor.NewHeightController()
				if _, ok := subWorkers[0].(actor.FSMCompatible); ok {
					previous = baseWorker.Next(actor.NewFSMController()).Next(getLinkedActor(subWorkers[0]))
				} else {
					previous = baseWorker.Next(getLinkedActor(subWorkers[0]))
				}
			} else {
				if _, ok := subWorkers[0].(actor.FSMCompatible); ok {
					baseWorker = actor.NewFSMController()
					previous = baseWorker.Next(getLinkedActor(subWorkers[0]))
				} else {
					baseWorker = getLinkedActor(subWorkers[0])
					previous = baseWorker
				}
			}

			for i := 1; i < len(subWorkers)-1; i++ {
				if _, ok := subWorkers[i].(actor.FSMCompatible); ok {
					previous = previous.Next(actor.NewFSMController()).Next(getLinkedActor(subWorkers[i]))
				} else {
					previous = previous.Next(getLinkedActor(subWorkers[i]))
				}
			}

			if _, ok := subWorkers[len(subWorkers)-1].(actor.FSMCompatible); ok {
				previous.Next(actor.NewFSMController()).EndWith(subWorkers[len(subWorkers)-1])
			} else {
				previous.EndWith(subWorkers[len(subWorkers)-1])
			}
			// config.createActor(name, baseWorker, broker)
			config.workersDict[name] = baseWorker
			config.createMsgOps(baseWorker, broker)
		} else {
			worker := config.createWorker(name, params, globalConfig)
			var baseWorker actor.LinkedActor
			if _, ok := worker.(actor.HeightSensitive); ok {
				baseWorker = actor.NewHeightController()
				if _, ok := worker.(actor.FSMCompatible); ok {
					baseWorker.Next(actor.NewFSMController()).EndWith(worker)
				} else {
					baseWorker.EndWith(worker)
				}
			} else {
				if _, ok := worker.(actor.FSMCompatible); ok {
					baseWorker = actor.NewFSMController()
					baseWorker.EndWith(worker)
				}
			}

			if baseWorker != nil {
				worker = baseWorker
			}
			// config.createActor(name, worker, broker)
			config.workersDict[name] = worker
			config.createMsgOps(worker, broker)
		}
	}

	for _, op := range config.MsgOps {
		switch op.Type {
		case "combine":
			inputs := make([]string, 0, len(op.Params.([]interface{})))
			for _, p := range op.Params.([]interface{}) {
				inputs = append(inputs, p.(string))
			}
			config.workersDict[actor.CombinedName(inputs...)] = actor.Combine(inputs...)
		case "rename":
			from := op.Params.([]interface{})[0].(string)
			to := op.Params.([]interface{})[1].(string)
			config.workersDict[actor.RenamerName(from, to)] = actor.Rename(from).To(to)
		default:
			panic("unknown operation type " + op.Type)
		}
	}

	for _, i := range config.Interfaces {
		srv := intf.Factory.Create(
			i.Service,
			globalConfig.GetConcurrency(config.Settings.ServiceName),
			config.Settings.ServiceName,
			i.Params,
		)
		intf.Router.Register(i.Name, srv, globalConfig.Rpc[i.Name], globalConfig.Zookeeper)
	}
	return config.workersDict
}

func (config *AppConfig) createMsgOps(worker actor.IWorkerEx, broker *streamer.StatefulStreamer) {
	inputs, _ := worker.Inputs()
	for _, input := range inputs {
		if strings.HasPrefix(input, actor.CombinerPrefix) {
			if _, ok := config.workersDict[input]; ok {
				continue
			}

			config.workersDict[input] = actor.Combine(strings.Split(input[len(actor.CombinerPrefix):], "-")...)
		}
	}
}

func (config *AppConfig) createActor(name string, worker actor.IWorkerEx, broker *streamer.StatefulStreamer) {
	config.workersDict[name] = worker
	workerActor := actor.NewActorEx(name, broker, worker)
	_, isConjunction := worker.Inputs()
	if isConjunction {
		workerActor.Connect(streamer.NewConjunctions(workerActor))
	} else {
		workerActor.Connect(streamer.NewDisjunctions(workerActor, 1))
	}
}

func (config *AppConfig) createWorker(name string, params map[string]interface{}, globalConfig GlobalConfig) actor.IWorkerEx {
	return actor.Factory.Create(
		name,
		globalConfig.GetConcurrency(config.Settings.ServiceName),
		config.Settings.ServiceName,
		config.replaceEnv(params, globalConfig),
	)
}

func getLinkedActor(worker actor.IWorkerEx) actor.LinkedActor {
	if _, ok := worker.(actor.LinkedActor); !ok {
		return actor.MakeLinkable(worker)
	}
	return worker.(actor.LinkedActor)
}

func (config *AppConfig) replaceEnv(params map[string]interface{}, globalConfig GlobalConfig) map[string]interface{} {
	for k, v := range params {
		if value, ok := v.(string); ok && value == "__env__" {
			params[k] = config.Settings.Env(k)
		} else if value == "__global__" {
			switch k {
			case "chain_id":
				params[k] = globalConfig.ChainId
			case "zookeeper":
				params[k] = globalConfig.Zookeeper
			case "coinbase":
				params[k] = globalConfig.Coinbase
			case "persistent_peers":
				params[k] = globalConfig.PersistentPeers
			case "remote_caches":
				params[k] = globalConfig.RemoteCaches
			case "cluster_name":
				params[k] = globalConfig.ClusterName
			case "p2p.peers":
				params[k] = globalConfig.P2pPeers
			case "p2p.gateway":
				params[k] = globalConfig.P2pGateway
			case "p2p.conn":
				params[k] = globalConfig.P2pConn
			default:
				panic("unsupport global variable")
			}
		}
	}
	return params
}

func PrintWorkers(workers map[string]actor.IWorkerEx) {
	for name, worker := range workers {
		fmt.Printf("%v { ", name)
		if _, ok := worker.(actor.FSMCompatible); ok {
			fmt.Print("FSMCompatible ")
		}
		if _, ok := worker.(actor.HeightSensitive); ok {
			fmt.Print("HeightSensitive ")
		}
		if _, ok := worker.(actor.Configurable); ok {
			fmt.Print("Configurable ")
		}
		if _, ok := worker.(actor.Initializer); ok {
			fmt.Print("Initializer ")
		}
		fmt.Print("}\n")

		inputs, isConjunction := worker.Inputs()
		fmt.Print("\tIN")
		if isConjunction {
			fmt.Print("[AND] { ")
		} else {
			fmt.Print("[OR] { ")
		}
		for _, input := range inputs {
			fmt.Printf("%v ", input)
		}
		outputs := worker.Outputs()
		fmt.Print("}, OUT { ")
		for output := range outputs {
			fmt.Printf("%v ", output)
		}
		fmt.Print("}\n")
	}
}

func GenerateDot(workers map[string]actor.IWorkerEx, kafkaConfig KafkaConfig, output string) {
	var g graph
	// Write header.
	index := 1
	port := 1
	inputPorts := make(map[string][]string)
	outputPorts := make(map[string][]string)
	// Write nodes.
	for name, worker := range workers {
		var n node
		// Write node header.
		inputs, isConjunction := worker.Inputs()
		var attr string
		if isConjunction {
			attr = "[AND]"
		} else {
			attr = "[OR]"
		}
		n.name = fmt.Sprintf("%s %s", name, attr)
		// Write inputs.
		for _, in := range inputs {
			_, topic := kafkaConfig.getServerTopic(in)
			if topic == "" {
				n.inputs = append(n.inputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, in))
				inputPorts[in] = append(inputPorts[in], fmt.Sprintf("thread%d:p%d", index, port))
				port++
			} else {
				n.inputs = append(n.inputs, fmt.Sprintf("<TD BGCOLOR=\"green\">%s</TD>", in))
			}
		}
		// Write outputs.
		outputs := worker.Outputs()
		for out := range outputs {
			_, topic := kafkaConfig.getServerTopic(out)
			if topic == "" {
				n.outputs = append(n.outputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, out))
				outputPorts[out] = append(outputPorts[out], fmt.Sprintf("thread%d:p%d", index, port))
				port++
			} else {
				n.outputs = append(n.outputs, fmt.Sprintf("<TD BGCOLOR=\"yellow\">%s</TD>", out))
			}
		}

		g.nodes = append(g.nodes, n)
		index++
	}
	// Write edges.
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

	writeDot(g, output)
}

type node struct {
	name    string
	inputs  []string
	outputs []string
}

type edge struct {
	from string
	to   string
}

type graph struct {
	nodes []node
	edges []edge
}

func writeDot(g graph, file string) {
	// Write header.
	text := "digraph arch {\n\trankdir=LR\n\tnode [shape=plaintext]\n"
	index := 1
	// Write nodes.
	for _, n := range g.nodes {
		// Write node header.
		text += fmt.Sprintf("\tthread%d [\n\t\tlabel=<\n\t\t<TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n\t\t\t<TR><TD COLSPAN=\"2\">%s</TD></TR>\n\t\t\t<TR>\n", index, n.name)
		// Write inputs.
		text += "\t\t\t\t<TD><TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
		for _, in := range n.inputs {
			text += fmt.Sprintf("\t\t\t\t<TR>%s</TR>\n", in)
		}
		if len(n.inputs) == 0 {
			text += "\t\t\t\t<TR><TD BGCOLOR=\"grey\">N/A</TD></TR>\n"
		}
		text += "\t\t\t\t</TABLE></TD>\n"
		// Write outputs.
		text += "\t\t\t\t<TD><TABLE BORDER=\"0\" CELLBORDER=\"1\" CELLSPACING=\"0\">\n"
		for _, out := range n.outputs {
			text += fmt.Sprintf("\t\t\t\t<TR>%s</TR>\n", out)
		}
		if len(n.outputs) == 0 {
			text += "\t\t\t\t<TR><TD BGCOLOR=\"grey\">N/A</TD></TR>\n"
		}
		text += "\t\t\t\t</TABLE></TD>\n"
		// Write node footer.
		text += "\t\t\t</TR>\n\t\t</TABLE>>\n\t]\n"

		index++
	}
	// Write edges.
	for _, e := range g.edges {
		text += fmt.Sprintf("\t%s -> %s\n", e.from, e.to)
	}
	// Write footer.
	text += "}\n"

	os.WriteFile(file, []byte(text), 0644)
}
