package p2p

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/main/modules/p2p/conn/status"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/config"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/server"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/slave"
	"github.com/go-zookeeper/zk"
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

	jsonStr, _ := json.Marshal(params["p2p.gateway"])
	json.Unmarshal(jsonStr, &gateway.cfg.Server)
	gateway.cfg.Server.ID = params["cluster_name"].(string)

	jsonStr, _ = json.Marshal(params["p2p.peers"])
	json.Unmarshal(jsonStr, &gateway.cfg.Peers)
}

func (gateway *P2pGateway) OnStart() {
	zkConn, _, err := zk.Connect(gateway.cfg.ZooKeeper.Servers, 3*time.Second)
	if err != nil {
		panic(err)
	}
	// defer zkConn.Close()

	server := server.NewServer(gateway.cfg, zkConn)
	go server.Serve()

	svcMonitor := slave.NewMonitor(zkConn, gateway.cfg.ZooKeeper.ConnStatusRoot, func(services map[string]*status.SvcStatus) {
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
