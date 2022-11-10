package server

import (
	"bufio"
	"fmt"
	"net"
	"time"

	"github.com/arcology-network/main/modules/p2p/conn/config"
	"github.com/arcology-network/main/modules/p2p/conn/peer"
	"github.com/arcology-network/main/modules/p2p/conn/protocol"
	"github.com/arcology-network/main/modules/p2p/conn/status"
	"github.com/go-zookeeper/zk"
)

type Server struct {
	cfg           *config.Config
	col           *status.Collector
	sw            *peer.Switch
	onMsgReceived func(string, *protocol.Message)
	whitelist     []*config.PeerConfig
	zkConn        *zk.Conn
}

func NewServer(cfg *config.Config, col *status.Collector, onMsgReceived func(string, *protocol.Message)) *Server {
	server := &Server{
		cfg:           cfg,
		col:           col,
		onMsgReceived: onMsgReceived,
	}

	var err error
	server.zkConn, _, err = zk.Connect(cfg.ZooKeeper.Servers, 30*time.Second)
	if err != nil {
		panic(err)
	}

	server.sw = peer.NewSwitch(
		cfg.Server.SID,
		onMsgReceived,
		func(id string) {
			fmt.Printf("[Server.OnPeerClosed] peer %s closed\n", id)
			path := cfg.ZooKeeper.PeerConfigRoot + "/" + id
			_, s, err := server.zkConn.Get(path)
			if err != nil {
				panic(err)
			}
			err = server.zkConn.Delete(path, s.Version)
			if err != nil {
				panic(err)
			}
		},
		col)

	return server
}

func (s *Server) Start() {
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
			// TODO: log error.
			continue
		}

		go s.handleClient(conn)
	}
}

func (s *Server) RefreshPeers(peers []*config.PeerConfig) {
	fmt.Printf("[conn.Server.RefreshPeers] peers:\n")
	for _, peer := range peers {
		fmt.Printf("\t%v\n", peer)
	}

	for _, cfg := range s.whitelist {
		exists := false
		for _, p := range peers {
			// TODO: more complicated rules needed.
			if cfg.ID == p.ID {
				exists = true
			}
		}

		if !exists {
			// Close this connection
		}
	}

	for _, p := range peers {
		exists := false
		for _, cfg := range s.whitelist {
			// TODO: more complicated rules needed.
			if cfg.ID == p.ID {
				exists = true
			}
		}

		if exists {
			continue
		}

		if p.Mode == config.WorkModePassive {
			// Do nothing.
		} else if p.Mode == config.WorkModeActive {
			fmt.Printf("[conn.Server.RefreshPeers] create new connections to peer %v\n", p)
			// Init new connection.
			newPeer := peer.NewPeer(s.cfg.Server.NID, p, s.onMsgReceived)
			connections, err := newPeer.Connect()
			if err != nil {
				panic(err)
			}

			for _, c := range connections {
				s.sw.AddConnection(p, c.GetConn())
			}
			s.sw.GetPeerReady(p.Host)
		} else {
			panic(fmt.Sprintf("invalid peer mode: %d", p.Mode))
		}
	}

	s.whitelist = peers
}

func (s *Server) Broadcast(msg *protocol.Message, pushIn bool) {
	s.sw.Broadcast(msg, pushIn)
}

func (s *Server) Send(peer string, msg *protocol.Message) error {
	s.sw.Send(peer, msg)
	return nil
}

func (s *Server) handleClient(conn net.Conn) {
	fmt.Printf("conn: %v\n", conn.RemoteAddr())
	r := bufio.NewReader(conn)

	m, err := protocol.ReadMessage(r)
	if err != nil {
		fmt.Printf("read message error: %v\n", err)
		return
	}

	if m.Type != protocol.MessageTypeHandshake {
		fmt.Printf("unknown message type %v\n", m.Type)
		return
	}

	peerCfg, err := s.handleMsg(conn, m)
	if err != nil {
		fmt.Printf("handle message error: %v\n", err)
		return
	}

	s.sw.AddConnection(peerCfg, conn)
}

func (s *Server) handleMsg(conn net.Conn, m *protocol.Message) (*config.PeerConfig, error) {
	var msg protocol.MsgHandshake
	ack, peerCfg, err := s.validatePeer(msg.FromMessage(m))
	if err != nil {
		return nil, err
	}

	if err := protocol.WriteMessage(conn, ack.ToMessage()); err != nil {
		return nil, err
	}

	fmt.Printf("handshake with %s success\n", msg.ID)
	return peerCfg, nil
}

func (s *Server) validatePeer(m *protocol.MsgHandshake) (*protocol.MsgHandshake, *config.PeerConfig, error) {
	var peerCfg *config.PeerConfig
	for _, peer := range s.whitelist {
		if m.ID == peer.ID {
			peerCfg = peer
		}
	}

	if peerCfg == nil {
		// FIXME
		fmt.Printf("[conn.Server.validatePeer] peer config not found, sleep 3 seconds and try again")
		time.Sleep(3 * time.Second)
		for _, peer := range s.whitelist {
			if m.ID == peer.ID {
				peerCfg = peer
			}
		}

		if peerCfg == nil {
			return nil, nil, fmt.Errorf("invalid peer %s", m.ID)
		}
	}

	// TODO

	ack := *m
	ack.ID = s.cfg.Server.NID
	return &ack, peerCfg, nil
}
