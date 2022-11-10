package status

import (
	"encoding/json"
	"sync"

	"github.com/arcology-network/main/modules/p2p/conn/config"
)

type Peer struct {
	ID          string             `json:"id"`
	Config      *config.PeerConfig `json:"config"`
	Connections []string           `json:"connections"`
}

type SvcStatus struct {
	SvcConfig *config.Config   `json:"config"`
	Peers     map[string]*Peer `json:"peers"`
	plock     sync.Mutex       `json:"-"`
}

func (s SvcStatus) ToJsonStr() (string, error) {
	b, err := json.Marshal(s)
	return string(b), err
}

func (s *SvcStatus) FromJsonStr(str string) error {
	return json.Unmarshal([]byte(str), s)
}

func (s *SvcStatus) AddPeer(id string, p *Peer) {
	s.plock.Lock()
	defer s.plock.Unlock()
	s.Peers[id] = p
}
