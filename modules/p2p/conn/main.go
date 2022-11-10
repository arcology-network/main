package main

import (
	"fmt"
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
			fmt.Printf("%v\n", cfg)
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
