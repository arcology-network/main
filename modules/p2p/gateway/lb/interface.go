package lb

import (
	"github.com/HPISTechnologies/main/modules/p2p/conn/status"
)

type LoadBalanceStrategy interface {
	GetSvcID(map[string]*status.SvcStatus, string, int) string
}
