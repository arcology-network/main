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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/actor/rpc"
	brokerpk "github.com/arcology-network/streamer/broker"
	jetlib "github.com/arcology-network/streamer/jet/lib"
	"github.com/arcology-network/streamer/logger"
	"github.com/spf13/viper"
)

var testRuntimePathKeys = map[string]struct{}{
	"dbpath":               {},
	"jwt_file":             {},
	"logfile":              {},
	"root":                 {},
	"storage_block_path":   {},
	"storage_index_path":   {},
	"storage_receipt_path": {},
	"storage_state_path":   {},
	"storage_tmblock_dir":  {},
	"tm_state_store_dir":   {},
}

func newTestRuntimeRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("", "arcology-main-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

func runtimeTestPath(root, key, configured string) string {
	configured = strings.ReplaceAll(configured, "\\", "/")
	parts := strings.FieldsFunc(configured, func(r rune) bool { return r == '/' })
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "." && part != ".." && part != "" {
			clean = append(clean, part)
		}
	}
	if len(clean) == 0 {
		clean = append(clean, key)
	}
	return filepath.Join(append([]string{root}, clean...)...)
}

func redirectTestRuntimePaths(value interface{}, root string) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			if _, ok := testRuntimePathKeys[key]; ok {
				if configured, ok := child.(string); ok && configured != "__env__" && configured != "__global__" {
					typed[key] = runtimeTestPath(root, key, configured)
					continue
				}
			}
			redirectTestRuntimePaths(child, root)
		}
	case []interface{}:
		for _, child := range typed {
			redirectTestRuntimePaths(child, root)
		}
	}
}

func loadTestConf(t *testing.T, globalConfigFile, jetConfigFile, appConfigFile string) {
	t.Helper()
	root := newTestRuntimeRoot(t)

	ss := jetlib.RunJetTestServer()
	globalConfig, err := LoadGlobalConfig(globalConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	jetConfig, err := jetlib.LoadConfig(jetConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	appConfig, err := LoadAppConfig(appConfigFile)
	if err != nil {
		t.Fatal(err)
	}

	redirectTestRuntimePaths(appConfig.Settings.Envs, root)
	for _, params := range appConfig.Actors {
		redirectTestRuntimePaths(params, root)
	}
	viper.Set("home", filepath.Join(root, "tmroot"))
	jetConfig.Nats.Servers[0] = ss.ClientURL()

	logPath := filepath.Join(root, "logs", "app.log")
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		t.Fatal(err)
	}
	logger.InitLog(globalConfig.LogConfigFile, logPath)

	broker := brokerpk.NewStatefulStreamer()
	rpc.InitGlobalRPCFactory()
	rpc.InitGlobalRPCClient(broker, globalConfig.RpcConcurrent, globalConfig.RpcTimeoutSeconds)
	workers := appConfig.InitApp(broker, globalConfig, jetConfig)
	broker.Serve()

	for _, worker := range workers {
		if initializer, ok := worker.(actor.Initializer); ok {
			for _, msg := range initializer.InitMsgs() {
				broker.Send(msg.Name, msg)
			}
		}
	}
	for i := range appConfig.StartMsgs {
		msg := &appConfig.StartMsgs[i]
		broker.Send(msg.Name, msg)
	}
}

func TestActorNode(t *testing.T) {
	appConfigFile := "../modules/pool/pool.yaml"
	appConfig, _ := LoadAppConfig(appConfigFile)

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
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/arbitrator/arbitrator.yaml")
}

func TestLoadConsensus(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/consensus/consensus.yaml")
}

func TestLoadCoordinator(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/coordinator/coordinator.yaml")
}

func TestLoadCore(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/core/core.yaml")
}

func TestLoadEthApi(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/eth-api/eth-api.yaml")
}

func TestLoadExec(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/exec/exec.yaml")
}

func TestLoadGateway(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/gateway/gateway.yaml")
}

func TestLoadPool(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/pool/pool.yaml")
}

func TestLoadReceiptHashing(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/receipt-hashing/receipt-hashing.yaml")
}

func TestLoadScheduler(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/scheduler/scheduler.yaml")
}

func TestLoadStorage(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/storage/storage.yaml")
}

func TestLoadTpp(t *testing.T) {
	loadTestConf(t, "./global.yaml", "./jet.yaml", "../modules/tpp/tpp.yaml")
}
