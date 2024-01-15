package core

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type Initializer struct {
	actor.WorkerThread

	inited bool
}

func NewInitializer(concurrency int, groupId string) actor.IWorkerEx {
	w := &Initializer{}
	w.Set(concurrency, groupId)
	return w
}

func (i *Initializer) Inputs() ([]string, bool) {
	return []string{actor.MsgBlockCompleted}, false
}

func (i *Initializer) Outputs() map[string]int {
	return map[string]int{
		actor.MsgLocalParentInfo: 1,
		actor.MsgParentInfo:      1,
	}
}

func (i *Initializer) OnStart() {}

func (i *Initializer) OnMessageArrived(msgs []*actor.Message) error {
	if !i.inited {
		var parentInfo cmntyp.ParentInfo
		iin := 12
		if err := intf.Router.Call("statestore", "GetParentInfo", &iin, &parentInfo); err != nil {
			panic(err)
		}
		//fmt.Printf("[core.Initializer.OnMessageArrived] init parent info: %v\n", parentInfo)
		i.MsgBroker.Send(actor.MsgLocalParentInfo, &parentInfo)
		i.MsgBroker.Send(actor.MsgParentInfo, &parentInfo)
		i.inited = true
	}
	return nil
}
