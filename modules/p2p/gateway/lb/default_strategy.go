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
