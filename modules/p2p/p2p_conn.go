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
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/peer"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
	"github.com/arcology-network/main/modules/p2p/conn/server"
	"github.com/arcology-network/main/modules/p2p/conn/status"
	"github.com/arcology-network/streamer/actor"
)

type P2pConn struct {
	actor.WorkerThread

	cfg           *config.Config
	broadcastChan chan *protocol.Message
	srv           *server.Server
	assembler     *peer.MessageAssembler
}

var (
	connInstance *P2pConn
	initOnce     sync.Once
)

func NewP2pConn(concurrency int, groupId string) actor.IWorkerEx {
	initOnce.Do(func() {
		connInstance = &P2pConn{
			cfg:           &config.Config{},
			broadcastChan: make(chan *protocol.Message, 10),
		}
		connInstance.assembler = peer.NewMessageAssembler(connInstance.broadcastChan, 10)
		connInstance.Set(concurrency, groupId)
	})

	return connInstance
}

func (conn *P2pConn) Inputs() ([]string, bool) {
	return []string{
		actor.MsgP2pSent,
	}, false
}

func (conn *P2pConn) Outputs() map[string]int {
	return map[string]int{
		actor.MsgP2pReceived: 1,
	}
}

func (conn *P2pConn) Config(params map[string]interface{}) {
	conn.cfg.ZooKeeper.Servers = []string{params["zookeeper"].(string)}
	conn.cfg.ZooKeeper.PeerConfigRoot = "/p2p/peer/config"
	conn.cfg.ZooKeeper.ConnStatusRoot = "/p2p/conn/status"

	jsonStr, _ := json.Marshal(params["p2p_conn"])
	json.Unmarshal(jsonStr, &conn.cfg.Server)
	conn.cfg.Server.NID = params["cluster_name"].(string)
	conn.cfg.Server.SID = "srv1"
}

func (conn *P2pConn) OnStart() {
	collector := status.NewCollector(conn.cfg, conn.cfg.ZooKeeper.Servers)
	collector.UpdateZKStatus()
	go collector.Start()

	conn.srv = server.NewServer(conn.cfg, collector, func(topic string, msg *protocol.Message) {
		// packages := msg.ToPackages()
		// fmt.Printf("[P2pConn.OnStart] %d packages received\n", len(packages))
		// for _, p := range packages {
		// 	b, _ := p.MarshalBinary()
		// 	conn.MsgBroker.Send(actor.MsgP2pReceived, b)
		// }
		conn.MsgBroker.Send(actor.MsgP2pReceived, msg)
	})
	go conn.srv.Start()
	go conn.assembler.Serve()

	go func() {
		for m := range conn.broadcastChan {
			fmt.Printf("[P2pConn.OnStart] in broadcast channel for loop\n")
			conn.srv.Broadcast(m, false)
		}
	}()

	watcher, err := config.NewPeerConfigWatcher(conn.cfg.ZooKeeper.Servers, conn.cfg.ZooKeeper.PeerConfigRoot, func(configs []*config.PeerConfig) {
		var peersToServe []*config.PeerConfig
		for _, cfg := range configs {
			fmt.Printf("%v\n", cfg)
			if cfg.AssignTo == conn.cfg.Server.SID {
				peersToServe = append(peersToServe, cfg)
			}
		}
		conn.srv.RefreshPeers(peersToServe)
	})
	if err != nil {
		panic(err)
	}
	go watcher.Serve()
}

func (conn *P2pConn) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgP2pSent:
		// fmt.Printf("[P2pConn.OnMessageArrived] broadcast message received: %v\n", msg)
		data := msg.Data.([]byte)
		var p protocol.Package
		p.UnmarshalBinary(data)
		p.Body = data[protocol.PackageHeaderSize:]

		if p.Header.TotalPackageCount == 1 {
			var m protocol.Message
			conn.broadcastChan <- m.FromPackages([]*protocol.Package{&p})
		} else {
			conn.assembler.AddPart(&p)
		}
	}
	return nil
}

type P2pSendRequest struct {
	Peer string
	Msg  *protocol.Message
}

func (conn *P2pConn) Send(ctx context.Context, request *P2pSendRequest, _ *int) error {
	fmt.Printf("[P2pConn] Send request via rpc, peer = %v\n", request.Peer)
	return conn.srv.Send(request.Peer, request.Msg)
}
