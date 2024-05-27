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

package lb

import (
	"sort"

	"github.com/arcology-network/main/modules/p2p/conn/status"
)

type DefaultStrategy struct {
}

func NewDefaultStrategy() *DefaultStrategy {
	return &DefaultStrategy{}
}

func (s *DefaultStrategy) GetSvcID(slaves map[string]*status.SvcStatus, peerID string, connectionCount int) string {
	var slaveList []*status.SvcStatus
	for _, slave := range slaves {
		slaveList = append(slaveList, slave)
	}
	sort.Slice(slaveList, func(i, j int) bool {
		return connectionsOnSlave(slaveList[i]) < connectionsOnSlave(slaveList[j])
	})

	// FIXME: this is tricky.
	slaveList[0].AddPeer(peerID, &status.Peer{
		Connections: make([]string, connectionCount),
	})
	return slaveList[0].SvcConfig.Server.SID
}

func connectionsOnSlave(status *status.SvcStatus) int {
	count := 0
	for _, p := range status.Peers {
		count += len(p.Connections)
	}
	return count
}
