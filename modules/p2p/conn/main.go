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
	"os"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/peer"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
	"github.com/arcology-network/main/modules/p2p/conn/receiver"
	"github.com/arcology-network/main/modules/p2p/conn/sender"
	"github.com/arcology-network/main/modules/p2p/conn/server"
	"github.com/arcology-network/main/modules/p2p/conn/status"
)

func main() {
	cfgFile := "config/config.yml"
	if len(os.Args) > 1 {
		cfgFile = os.Args[1]
	}

	serverCfg, err := config.LoadConfig(cfgFile)
	if err != nil {
		panic(err)
	}

	collector := status.NewCollector(serverCfg, serverCfg.ZooKeeper.Servers)
	collector.UpdateZKStatus()
	go collector.Start()

	sender, err := sender.NewKafkaSender(serverCfg.Kafka.Servers, serverCfg.Kafka.TopicIn)
	if err != nil {
		panic(err)
	}
	srv := server.NewServer(serverCfg, collector, func(topic string, msg *protocol.Message) {
		packages := msg.ToPackages()
		for _, p := range packages {
			b, _ := p.MarshalBinary()
			sender.Send(b)
		}
	})
	go srv.Start()

	receivedMsg := make(chan *protocol.Message, 10)
	assembler := peer.NewMessageAssembler(receivedMsg, 10)
	go assembler.Serve()
	receiver := receiver.NewKafkaReceiver(
		serverCfg.Kafka.Servers,
		[]string{serverCfg.Kafka.TopicOut},
		serverCfg.Server.SID,
		func(topic string, data []byte) {
			var p protocol.Package
			p.UnmarshalBinary(data)
			p.Body = data[protocol.PackageHeaderSize:]

			if p.Header.TotalPackageCount == 1 {
				var m protocol.Message
				receivedMsg <- m.FromPackages([]*protocol.Package{&p})
			} else {
				assembler.AddPart(&p)
			}
		},
	)
	receiver.Start()

	go func() {
		for m := range receivedMsg {
			srv.Broadcast(m, false)
		}
	}()

	watcher, err := config.NewPeerConfigWatcher(serverCfg.ZooKeeper.Servers, serverCfg.ZooKeeper.PeerConfigRoot, func(configs []*config.PeerConfig) {
		var peersToServe []*config.PeerConfig
		for _, cfg := range configs {

			if cfg.AssignTo == serverCfg.Server.SID {
				peersToServe = append(peersToServe, cfg)
			}
		}
		srv.RefreshPeers(peersToServe)
	})
	if err != nil {
		panic(err)
	}
	watcher.Serve()
}
