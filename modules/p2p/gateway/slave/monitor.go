package slave

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/HPISTechnologies/main/modules/p2p/conn/status"
	"github.com/go-zookeeper/zk"
)

type Monitor struct {
	zkConn         *zk.Conn
	root           string
	services       map[string]*status.SvcStatus
	onStatusUpdate func(map[string]*status.SvcStatus)
	eventCh        <-chan zk.Event
}

func NewMonitor(zkConn *zk.Conn, root string, onStatusUpdate func(map[string]*status.SvcStatus)) *Monitor {
	m := &Monitor{
		zkConn:         zkConn,
		root:           root,
		services:       make(map[string]*status.SvcStatus),
		onStatusUpdate: onStatusUpdate,
	}

	var err error
	_, _, m.eventCh, err = m.zkConn.ChildrenW(m.root)
	if err != nil {
		panic(err)
	}
	return m
}

func (m *Monitor) Serve() {
	m.reloadSrvStatus()
	m.onStatusUpdate(m.services)

	for {
		e := <-m.eventCh
		var err error
		_, _, m.eventCh, err = m.zkConn.ChildrenW(m.root)
		if err != nil {
			panic(err)
		}

		if e.Type == zk.EventNodeChildrenChanged {
			m.reloadSrvStatus()
			m.onStatusUpdate(m.services)
		} else {
			fmt.Printf("[p2p.gateway] Unknown event type got %v\n", e.Type)
			time.Sleep(1 * time.Second)
		}
	}
}

func (m *Monitor) reloadSrvStatus() {
	m.services = make(map[string]*status.SvcStatus)
	children, _, err := m.zkConn.Children(m.root)
	if err != nil {
		panic(err)
	}

	for _, child := range children {
		fullpath := filepath.Join(m.root, child)
		data, _, err := m.zkConn.Get(fullpath)
		if err != nil {
			continue
		}

		var srvStatus status.SvcStatus
		err = srvStatus.FromJsonStr(string(data))
		if err != nil {
			continue
		}
		m.services[srvStatus.SvcConfig.Server.SID] = &srvStatus
	}
}
