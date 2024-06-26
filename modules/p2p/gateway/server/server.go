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

package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	cconfig "github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/connection"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
	"github.com/arcology-network/main/modules/p2p/conn/status"
	gconfig "github.com/arcology-network/main/modules/p2p/gateway/config"
	"github.com/arcology-network/main/modules/p2p/gateway/lb"
	"github.com/go-zookeeper/zk"
)

type Server struct {
	cfg       *gconfig.Config
	zkServers []string
	zkConn    *zk.Conn
	slaves    map[string]*status.SvcStatus
	slock     sync.RWMutex
}

func NewServer(cfg *gconfig.Config, zkServers []string) *Server {
	zkConn, _, err := zk.Connect(zkServers, 30*time.Second)
	if err != nil {
		panic(err)
	}

	return &Server{
		cfg:       cfg,
		zkServers: zkServers,
		zkConn:    zkConn,
	}
}

func (s *Server) OnSvcStatusUpdate(services map[string]*status.SvcStatus) {
	// If no conn-svc available, we can do nothing.
	if len(services) == 0 {
		return
	}
	s.slock.Lock()
	defer s.slock.Unlock()

	for _, peer := range s.cfg.Peers {
		found := false
		for _, status := range services {
			if _, ok := status.Peers[peer.ID]; ok {
				found = true
				break
			}
		}

		if !found {
			go func(host string, port int) {
				conn, err := connection.Connect(s.cfg.Server.ID, &cconfig.PeerConfig{
					Host:            host,
					Port:            port,
					ConnectionCount: 4,
				})
				if err != nil {
					fmt.Printf("connection error to %s: %s\n", host, err)
					return
				}
				defer conn.Close()

				peerCfg, err := conn.Handshake()
				if err != nil {
					fmt.Printf("handshake error to %s: %s\n", host, err)
					return
				}

				svcID := lb.NewDefaultStrategy().GetSvcID(s.slaves, peerCfg.ID, int(peerCfg.ConnectionCount))
				routingAck, err := conn.Routing(s.slaves[svcID].SvcConfig.Server.Host, s.slaves[svcID].SvcConfig.Server.Port)
				if err != nil {
					fmt.Printf("routing error to %s: %s\n", host, err)
					return
				}

				err = s.addPeerConfig(&cconfig.PeerConfig{
					ID:              peerCfg.ID,
					Host:            routingAck.Host,
					Port:            routingAck.Port,
					ConnectionCount: peerCfg.ConnectionCount,
					AssignTo:        svcID,
					Mode:            cconfig.WorkModeActive,
				})
				if err != nil {
					fmt.Printf("routing error to %s: %s\n", host, err)
				}
			}(peer.Host, peer.Port)
		}
	}
	s.slaves = services
}

func (s *Server) Serve() {
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", "0.0.0.0", s.cfg.Server.Port))
	if err != nil {
		panic(err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go s.handleClient(conn)
	}
}

func (s *Server) handleClient(conn net.Conn) {
	fmt.Printf("conn: %v\n", conn.RemoteAddr())
	defer conn.Close()
	r := bufio.NewReader(conn)

	m, err := protocol.ReadMessage(r)
	if err != nil {
		fmt.Printf("read handshake message error: %v\n", err)
		return
	}

	if m.Type != protocol.MessageTypeHandshake {
		fmt.Printf("unknown message type %v\n", m.Type)
		return
	}

	var handshake protocol.MsgHandshake
	handshakeAck, err := s.validatePeer(handshake.FromMessage(m))
	if err != nil {
		fmt.Printf("validate peer error: %v\n", err)
		return
	}

	if err := protocol.WriteMessage(conn, handshakeAck.ToMessage()); err != nil {
		fmt.Printf("write handshake message error: %v\n", err)
		return
	}

	fmt.Printf("handshake with %s success\n", handshake.ID)
	m, err = protocol.ReadMessage(r)
	if err != nil {
		fmt.Printf("read routing message error: %v\n", err)
		return
	}

	if m.Type != protocol.MessageTypeRouting {
		fmt.Printf("unknown message type %v\n", m.Type)
		return
	}

	var routing protocol.MsgRouting
	routingAck, err := s.routing(&handshake, routing.FromMessage(m))
	if err != nil {
		fmt.Printf("routing error: %v\n", err)
		return
	}

	if err = protocol.WriteMessage(conn, routingAck.ToMessage()); err != nil {
		fmt.Printf("write routing message error: %v\n", err)
		return
	}

	fmt.Printf("routing success, peer: %v, self: %v\n", routing, routingAck)
}

func (s *Server) validatePeer(m *protocol.MsgHandshake) (*protocol.MsgHandshake, error) {
	for _, peer := range s.cfg.Peers {
		if m.ID == peer.ID {
			ack := *m
			ack.ID = s.cfg.Server.ID
			return &ack, nil
		}
	}

	return nil, fmt.Errorf("invalid peer %s", m.ID)
}

func (s *Server) routing(handshake *protocol.MsgHandshake, routing *protocol.MsgRouting) (*protocol.MsgRouting, error) {
	s.slock.Lock()
	defer s.slock.Unlock()

	if len(s.slaves) == 0 {
		return nil, errors.New("no conn-svc available")
	}

	svcID := lb.NewDefaultStrategy().GetSvcID(s.slaves, handshake.ID, int(handshake.ConnectionCount))
	err := s.addPeerConfig(&cconfig.PeerConfig{
		ID:              handshake.ID,
		Host:            routing.Host,
		Port:            routing.Port,
		ConnectionCount: handshake.ConnectionCount,
		AssignTo:        svcID,
		Mode:            cconfig.WorkModePassive,
	})
	if err != nil {
		return nil, err
	}

	return &protocol.MsgRouting{
		Host: s.slaves[svcID].SvcConfig.Server.Host,
		Port: s.slaves[svcID].SvcConfig.Server.Port,
	}, nil
}

func (s *Server) addPeerConfig(peerCfg *cconfig.PeerConfig) error {
	json, err := peerCfg.ToJsonStr()
	if err != nil {
		return err
	}

	path := s.cfg.ZooKeeper.PeerConfigRoot + "/" + peerCfg.ID
	_, err = s.zkConn.Create(path, []byte(json), 0, zk.WorldACL(zk.PermAll))
	if err != nil {
		s.zkConn, _, err = zk.Connect(s.zkServers, 30*time.Second)
		if err != nil {
			return err
		}

		_, err = s.zkConn.Create(path, []byte(json), 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
	}

	return nil
}
