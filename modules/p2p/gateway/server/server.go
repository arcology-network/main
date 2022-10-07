package server

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"sync"

	cconfig "github.com/HPISTechnologies/main/modules/p2p/conn/config"
	"github.com/HPISTechnologies/main/modules/p2p/conn/connection"
	"github.com/HPISTechnologies/main/modules/p2p/conn/protocol"
	"github.com/HPISTechnologies/main/modules/p2p/conn/status"
	gconfig "github.com/HPISTechnologies/main/modules/p2p/gateway/config"
	"github.com/HPISTechnologies/main/modules/p2p/gateway/lb"
	"github.com/go-zookeeper/zk"
)

type Server struct {
	cfg    *gconfig.Config
	zkConn *zk.Conn
	slaves map[string]*status.SvcStatus
	slock  sync.RWMutex
}

func NewServer(cfg *gconfig.Config, zkConn *zk.Conn) *Server {
	return &Server{
		cfg:    cfg,
		zkConn: zkConn,
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
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port))
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

	fmt.Printf(fmt.Sprintf("handshake with %s success\n", handshake.ID))
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

	fmt.Printf(fmt.Sprintf("routing success, peer: %v, self: %v\n", routing, routingAck))
}

func (s *Server) validatePeer(m *protocol.MsgHandshake) (*protocol.MsgHandshake, error) {
	for _, peer := range s.cfg.Peers {
		if m.ID == peer.ID {
			ack := *m
			ack.ID = s.cfg.Server.ID
			return &ack, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("invalid peer %s", m.ID))
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
		return err
	}

	return nil
}
