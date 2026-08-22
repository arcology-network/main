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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/arcology-network/main/components/storage"
	_ "github.com/arcology-network/main/modules"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/actor/rpc"
	_ "github.com/arcology-network/streamer/aggregator/v3"
	brokerpk "github.com/arcology-network/streamer/broker"
	"github.com/arcology-network/streamer/jet"
	jetlib "github.com/arcology-network/streamer/jet/lib"
	"github.com/arcology-network/streamer/logger"
	"gopkg.in/yaml.v2"
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
	Settings   Settings                          `yaml:"settings"`
	Actors     map[string]map[string]interface{} `yaml:"actors"`
	MsgOps     []MsgOperation                    `yaml:"msg_ops"`
	Interfaces []Interface                       `yaml:"interfaces"`
	StartMsgs  []actor.Message                   `yaml:"start_msgs"`

	WorkersDict map[string]actor.Business
	actorDict   map[string]*actor.Actor
}

func LoadAppConfig(file string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	cfg := &AppConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	cfg.Settings.ServiceName = strings.Split(filepath.Base(file), ".")[0]
	cfg.WorkersDict = make(map[string]actor.Business)

	cfg.actorDict = make(map[string]*actor.Actor)

	return cfg, nil
}

func (config *AppConfig) InitApp(broker *brokerpk.StatefulStreamer, globalConfig *GlobalConfig, jetConfig *jetlib.JetStreamConfig) map[string]actor.Business {
	// inputs := make([][]string, 0, 2*len(config.Actors))
	outputs := make([]map[string]int, 0, 2*len(config.Actors))
	nameDic := map[string][]string{}
	actorTree := ParseActors(config.Actors)

	sender := actor.NewSendAdaptor(broker, rpc.GlobalRPCClient)

	for name, actorNode := range actorTree {
		params := actorNode.Params
		if _, ok := config.WorkersDict[name]; ok {
			continue
		}

		nameDic[name] = []string{}

		if name[0] == '-' {
			businessList := make([]actor.Business, len(actorNode.Subs))
			rpcSrvList := make([]string, len(actorNode.Subs))
			idx := 0
			for subname, actorNode := range actorNode.Subs {
				nameDic[name] = append(nameDic[name], subname)
				subparams := actorNode.Params
				if subname == "executor" {
					idx := subparams["idx"].(int)
					subparams = MergeMap(subparams, globalConfig.Executors[idx])
					rpcSrvList[idx] = subparams["name"].(string)
				} else {
					rpcSrvList[idx] = ""
				}

				businessList[idx] = config.createWorker(subname, subparams, globalConfig, sender)
				config.WorkersDict[subname] = businessList[idx]
				outputs = append(outputs, businessList[idx].Outputs())

				config.createMsgOps(businessList[idx], broker)

				idx++
			}

			config.actorDict[name] = actor.CreateActor(name, broker, businessList, nameDic[name], globalConfig.Concurrency, rpcSrvList)

		} else {
			rpcSrvList := []string{""}
			if name == "executor" {
				idx := params["idx"].(int)
				params = MergeMap(params, globalConfig.Executors[idx])
				rpcSrvList[0] = params["name"].(string)

			}
			nameDic[name] = []string{name}
			worker := config.createWorker(name, params, globalConfig, sender)
			config.WorkersDict[name] = worker
			outputs = append(outputs, worker.Outputs())
			config.createMsgOps(worker, broker)
			config.actorDict[name] = actor.CreateActor(name, broker, []actor.Business{worker}, []string{name}, globalConfig.Concurrency, rpcSrvList)
		}
	}

	for _, op := range config.MsgOps {
		switch op.Type {
		case "combine":
			inputs := make([]string, 0, len(op.Params.([]interface{})))
			for _, p := range op.Params.([]interface{}) {
				inputs = append(inputs, p.(string))
			}
			busi, act := actor.Combine(inputs...).On(broker)
			config.WorkersDict[actor.CombinedName(inputs...)] = busi
			config.actorDict[actor.CombinedName(inputs...)] = act
		case "rename":
			from := op.Params.([]interface{})[0].(string)
			to := op.Params.([]interface{})[1].(string)
			busi, act := actor.Rename(from).To(to).On(broker)
			config.WorkersDict[actor.RenamerName(from, to)] = busi
			config.actorDict[actor.RenamerName(from, to)] = act
		default:
			panic("unknown operation type " + op.Type)
		}
	}

	// inputsAll := actor.DeduplicationInputs(inputs)
	outputsALl := actor.DeduplicationOutputs(outputs)

	for actname, act := range config.actorDict {
		needCleaner := make(map[string]struct{})
		businessNames := nameDic[actname]
		haveConjunction := false
		for _, businessName := range businessNames {
			business := config.WorkersDict[businessName]
			ins, isConjunction := business.Inputs()
			if isConjunction {
				haveConjunction = true
			}
			for _, input := range ins {
				if !jetConfig.Contain(input) {
					continue
				}
				if _, ok := outputsALl[input]; ok {
					needCleaner[input] = struct{}{}
				}
			}

		}
		if len(needCleaner) != 0 {
			if haveConjunction {
				panic("Conjunction not supported.")
			}

			var msgs []string
			for msg := range needCleaner {
				msgs = append(msgs, msg)
			}

			filter := actor.NewOriginFilter(
				"origin-filter",
				actor.MsgsOnlyFrom(msgs, "downloader"),
			)
			act.SetFilters([]*actor.Filter{filter})
		}
	}

	jsm, err := jetlib.NewJetKVStreamerFromConfig(jetConfig)
	if err != nil {
		panic("create Jet Stream Err:" + err.Error())
	}

	if len(jetConfig.TopicsD) > 0 {
		downloader := jet.NewJetDownloader(jsm)
		config.SetSender(downloader, sender)
		actor.CreateActor("downloader", broker, []actor.Business{downloader}, []string{"downloader"}, globalConfig.Concurrency, []string{""})
		config.WorkersDict["downloader"] = downloader
	}

	if len(jetConfig.TopicsU) > 0 {
		uploader := jet.NewKafkaUploader(jsm)
		act := actor.CreateActor("uploader", broker, []actor.Business{uploader}, []string{"uploader"}, globalConfig.Concurrency, []string{""})
		filter := actor.NewOriginFilter(
			"origin-filter",
			actor.NotFrom("downloader"),
		)
		act.SetFilters([]*actor.Filter{filter})
		config.WorkersDict["uploader"] = uploader
	}

	return config.WorkersDict
}

func MergeMap(first, second map[string]interface{}) map[string]interface{} {
	for k, v := range second {
		first[k] = v
	}
	return first
}

func (config *AppConfig) createMsgOps(worker actor.Business, broker *brokerpk.StatefulStreamer) {
	inputs, _ := worker.Inputs()
	for _, input := range inputs {
		if strings.HasPrefix(input, actor.CombinerPrefix) {
			if _, ok := config.WorkersDict[input]; ok {
				continue
			}

			busi, act := actor.Combine(strings.Split(input[len(actor.CombinerPrefix):], "-")...).On(broker)
			config.WorkersDict[input] = busi
			config.actorDict[input] = act
		}
	}
}

func (config *AppConfig) SetSender(worker actor.Business, sender actor.OutboundSender) {
	if _, ok := worker.(actor.Sendable); ok {
		worker.(actor.Sendable).SetSender(sender)
	}
}

func (config *AppConfig) createWorker(name string, params map[string]interface{}, globalConfig *GlobalConfig, sender actor.OutboundSender) actor.Business {
	worker := actor.Factory.Create(
		name,
		globalConfig.Concurrency,
		config.Settings.ServiceName,
		config.replaceEnv(params, globalConfig),
		sender,
	)
	return worker
}

func (config *AppConfig) replaceEnv(params map[string]interface{}, globalConfig *GlobalConfig) map[string]interface{} {
	for k, v := range params {
		if value, ok := v.(string); ok && value == "__env__" {
			params[k] = config.Settings.Env(k)
		} else if value == "__global__" {
			switch k {
			case "chain_id":
				params[k] = globalConfig.ChainId
			case "cluster_name":
				params[k] = globalConfig.ClusterName
			case "executors":
				params[k] = globalConfig.Executors
			case "persistent_peers":
				params[k] = globalConfig.PersistentPeers
			default:
				panic("unsupport global variable:" + k)
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

func LoadConf(globalConfigFile, jetConfigFile, appConfigFile string) {
	ss := jetlib.RunJetTestServer()

	globalConfig, _ := LoadGlobalConfig(globalConfigFile)
	jetConfig, _ := jetlib.LoadConfig(jetConfigFile)
	appConfig, _ := LoadAppConfig(appConfigFile)

	jetConfig.Nats.Servers[0] = ss.ClientURL()

	logger.InitLog(globalConfig.LogConfigFile, "")

	broker := brokerpk.NewStatefulStreamer()
	rpc.InitGlobalRPCFactory()
	rpc.InitGlobalRPCClient(broker, globalConfig.RpcConcurrent, globalConfig.RpcTimeoutSeconds)

	workers := appConfig.InitApp(broker, globalConfig, jetConfig)

	broker.Serve()

	for _, worker := range workers {
		if _, ok := worker.(actor.Initializer); ok {
			msgs := worker.(actor.Initializer).InitMsgs()
			for _, msg := range msgs {
				broker.Send(msg.Name, msg)
			}
		}
	}

	for i := range appConfig.StartMsgs {
		msg := &appConfig.StartMsgs[i]
		broker.Send(msg.Name, msg)
	}

}

// func GenerateDot(workers map[string]actor.IWorkerEx, kafkaConfig KafkaConfig, output string) {
// 	var g graph
// 	// Write header.
// 	index := 1
// 	port := 1
// 	inputPorts := make(map[string][]string)
// 	outputPorts := make(map[string][]string)
// 	// Write nodes.
// 	for name, worker := range workers {
// 		var n node
// 		// Write node header.
// 		inputs, isConjunction := worker.Inputs()
// 		var attr string
// 		if isConjunction {
// 			attr = "[AND]"
// 		} else {
// 			attr = "[OR]"
// 		}
// 		n.name = fmt.Sprintf("%s %s", name, attr)
// 		// Write inputs.
// 		for _, in := range inputs {
// 			_, topic := kafkaConfig.getServerTopic(in)
// 			if topic == "" {
// 				n.inputs = append(n.inputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, in))
// 				inputPorts[in] = append(inputPorts[in], fmt.Sprintf("thread%d:p%d", index, port))
// 				port++
// 			} else {
// 				n.inputs = append(n.inputs, fmt.Sprintf("<TD BGCOLOR=\"green\">%s</TD>", in))
// 			}
// 		}
// 		// Write outputs.
// 		outputs := worker.Outputs()
// 		for out := range outputs {
// 			_, topic := kafkaConfig.getServerTopic(out)
// 			if topic == "" {
// 				n.outputs = append(n.outputs, fmt.Sprintf("<TD PORT=\"p%d\">%s</TD>", port, out))
// 				outputPorts[out] = append(outputPorts[out], fmt.Sprintf("thread%d:p%d", index, port))
// 				port++
// 			} else {
// 				n.outputs = append(n.outputs, fmt.Sprintf("<TD BGCOLOR=\"yellow\">%s</TD>", out))
// 			}
// 		}

// 		g.nodes = append(g.nodes, n)
// 		index++
// 	}
// 	// Write edges.
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

// 	writeDot(g, output)
// }

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
