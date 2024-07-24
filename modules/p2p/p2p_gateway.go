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

package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/arcology-network/main/modules/p2p/conn/status"
	"github.com/arcology-network/main/modules/p2p/gateway/config"
	"github.com/arcology-network/main/modules/p2p/gateway/server"
	"github.com/arcology-network/main/modules/p2p/gateway/slave"
	"github.com/arcology-network/streamer/actor"
)

type P2pGateway struct {
	actor.WorkerThread

	cfg *config.Config
}

func NewP2pGateway(concurrency int, groupId string) actor.IWorkerEx {
	w := &P2pGateway{
		cfg: &config.Config{},
	}
	w.Set(concurrency, groupId)
	return w
}

func (gateway *P2pGateway) Inputs() ([]string, bool) {
	return []string{}, false
}

func (gateway *P2pGateway) Outputs() map[string]int {
	return map[string]int{}
}

func (gateway *P2pGateway) Config(params map[string]interface{}) {
	gateway.cfg.ZooKeeper.Servers = []string{params["zookeeper"].(string)}
	gateway.cfg.ZooKeeper.PeerConfigRoot = "/p2p/peer/config"
	gateway.cfg.ZooKeeper.ConnStatusRoot = "/p2p/conn/status"

	jsonStr, _ := json.Marshal(params["p2p_gateway"])
	json.Unmarshal(jsonStr, &gateway.cfg.Server)
	gateway.cfg.Server.ID = params["cluster_name"].(string)

	jsonStr, _ = json.Marshal(params["p2p_peers"])
	json.Unmarshal(jsonStr, &gateway.cfg.Peers)
}

func (gateway *P2pGateway) OnStart() {
	server := server.NewServer(gateway.cfg, gateway.cfg.ZooKeeper.Servers)
	go server.Serve()

	svcMonitor := slave.NewMonitor(gateway.cfg.ZooKeeper.Servers, gateway.cfg.ZooKeeper.ConnStatusRoot, func(services map[string]*status.SvcStatus) {
		fmt.Printf("service status updated\n")
		for id, status := range services {
			fmt.Printf("\tid: %s, config: %v, peers: %v\n", id, status.SvcConfig, status.Peers)
		}
		server.OnSvcStatusUpdate(services)
	})
	go svcMonitor.Serve()
}

func (gateway *P2pGateway) OnMessageArrived(msgs []*actor.Message) error {
	return nil
}
