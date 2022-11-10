package status

import (
	"errors"
	"fmt"
	"time"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/go-zookeeper/zk"
)

type Event struct {
	RemoteAddr string
	Type       byte
	Data       interface{}
}

const (
	EventTypeNewConnection = 1
	EventTypePeerClosed    = 2
)

type Collector struct {
	path      string
	svcStatus *SvcStatus
	eventChan chan Event
	zkServers []string
	zkConn    *zk.Conn
}

func NewCollector(svcConfig *config.Config, zkServers []string) *Collector {
	zkConn, _, err := zk.Connect(zkServers, 30*time.Second)
	if err != nil {
		panic(err)
	}

	return &Collector{
		svcStatus: &SvcStatus{
			SvcConfig: svcConfig,
			Peers:     make(map[string]*Peer),
		},
		eventChan: make(chan Event, 100),
		zkServers: zkServers,
		zkConn:    zkConn,
	}
}

func (c *Collector) Start() {
	for event := range c.eventChan {
		peerID := event.Data.(*config.PeerConfig).ID
		switch event.Type {
		case EventTypeNewConnection:
			if _, ok := c.svcStatus.Peers[peerID]; !ok {
				c.svcStatus.Peers[peerID] = &Peer{
					ID:     event.Data.(*config.PeerConfig).ID,
					Config: event.Data.(*config.PeerConfig),
				}
			}
			c.svcStatus.Peers[peerID].Connections = append(c.svcStatus.Peers[peerID].Connections, event.RemoteAddr)
		case EventTypePeerClosed:
			delete(c.svcStatus.Peers, peerID)
		default:
			panic(fmt.Sprintf("unknown event type: %d", event.Type))
		}
		c.UpdateZKStatus()
		c.Print()
	}
}

func (c *Collector) Notify(event Event) {
	c.eventChan <- event
}

func (c *Collector) Print() {
	for id, peer := range c.svcStatus.Peers {
		fmt.Printf("Peer: id = %v, cfg = %v\n", id, peer.Config)
		for _, conn := range peer.Connections {
			fmt.Printf("\tconnection: addr = %v\n", conn)
		}
	}
}

func (c *Collector) UpdateZKStatus() {
	status, err := c.svcStatus.ToJsonStr()
	if err != nil {
		panic(err)
	}

	if c.path == "" {
		c.createPath(status)
	} else {
		fmt.Printf("[Collector.UpdateZKStatus] update path: %v\n", c.path)
		_, s, err := c.zkConn.Get(c.path)
		if errors.Is(err, zk.ErrConnectionClosed) {
			c.zkConn, _, err = zk.Connect(c.zkServers, 30*time.Second)
			if err != nil {
				panic(err)
			}

			_, s, err = c.zkConn.Get(c.path)
		}

		if err != nil {
			fmt.Printf("[Collector.UpdateZKStatus] failed to get path: %s\n", c.path)
			c.createPath(status)
		} else {
			_, err = c.zkConn.Set(c.path, []byte(status), s.Version)
			if err != nil {
				panic(err)
			}
		}
	}
}

func (c *Collector) createPath(status string) (err error) {
	path := c.svcStatus.SvcConfig.ZooKeeper.ConnStatusRoot + "/" + c.svcStatus.SvcConfig.Server.SID
	c.path, err = c.zkConn.CreateProtectedEphemeralSequential(path, []byte(status), zk.WorldACL(zk.PermAll))
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Collector.createPath] create ephemeral path: %s\n", c.path)
	return
}
