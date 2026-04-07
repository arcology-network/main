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
