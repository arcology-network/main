package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/main/modules/p2p/conn/config"
	"github.com/HPISTechnologies/main/modules/p2p/conn/peer"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
	"github.com/HPISTechnologies/main/modules/p2p/conn/server"
	"github.com/HPISTechnologies/main/modules/p2p/conn/status"
	"github.com/go-zookeeper/zk"
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

	jsonStr, _ := json.Marshal(params["p2p.conn"])
	json.Unmarshal(jsonStr, &conn.cfg.Server)
	conn.cfg.Server.NID = params["cluster_name"].(string)
	conn.cfg.Server.SID = "srv1"
}

func (conn *P2pConn) OnStart() {
	zkConn, _, err := zk.Connect(conn.cfg.ZooKeeper.Servers, 3*time.Second)
	if err != nil {
		panic(err)
	}
	// defer zkConn.Close()

	collector := status.NewCollector(conn.cfg, zkConn)
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

	watcher, err := config.NewPeerConfigWatcher(zkConn, conn.cfg.ZooKeeper.PeerConfigRoot, func(configs []*config.PeerConfig) {
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
