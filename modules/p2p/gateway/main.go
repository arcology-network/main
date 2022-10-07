package main

import (
	"fmt"
	"os"
	"time"

	"github.com/HPISTechnologies/main/modules/p2p/conn/status"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/config"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/server"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/slave"
	"github.com/go-zookeeper/zk"
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

	zkConn, _, err := zk.Connect(cfg.ZooKeeper.Servers, 3*time.Second)
	if err != nil {
		panic(err)
	}
	defer zkConn.Close()

	server := server.NewServer(cfg, zkConn)
	go server.Serve()

	svcMonitor := slave.NewMonitor(zkConn, cfg.ZooKeeper.ConnStatusRoot, func(services map[string]*status.SvcStatus) {
		fmt.Printf("service status updated\n")
		for id, status := range services {
			fmt.Printf("\tid: %s, config: %v, peers: %v\n", id, status.SvcConfig, status.Peers)
		}
		server.OnSvcStatusUpdate(services)
	})
	svcMonitor.Serve()
}
