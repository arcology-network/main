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

package main

import (
	"fmt"
	"os"

	"github.com/arcology-network/main/modules/p2p/conn/status"
	"github.com/arcology-network/main/modules/p2p/gateway/config"
	"github.com/arcology-network/main/modules/p2p/gateway/server"
	"github.com/arcology-network/main/modules/p2p/gateway/slave"
)

func main() {
	cfgFile := "config/config.yml"
	if len(os.Args) > 1 {
		cfgFile = os.Args[1]
	}

	cfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		panic(err)
	}

	server := server.NewServer(cfg, cfg.ZooKeeper.Servers)
	go server.Serve()

	svcMonitor := slave.NewMonitor(cfg.ZooKeeper.Servers, cfg.ZooKeeper.ConnStatusRoot, func(services map[string]*status.SvcStatus) {
		fmt.Printf("service status updated\n")
		for id, status := range services {
			fmt.Printf("\tid: %s, config: %v, peers: %v\n", id, status.SvcConfig, status.Peers)
		}
		server.OnSvcStatusUpdate(services)
	})
	svcMonitor.Serve()
}
