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
