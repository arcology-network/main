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
