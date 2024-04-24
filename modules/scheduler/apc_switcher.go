package scheduler

import (
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type ApcSwitcher struct {
	actor.WorkerThread
}

func NewApcSwitcher(concurrency int, groupId string) actor.IWorkerEx {
	apcSwitcher := &ApcSwitcher{}
	apcSwitcher.Set(concurrency, groupId)
	return apcSwitcher
}

func (r *ApcSwitcher) Inputs() ([]string, bool) {
	return []string{
		actor.MsgApcHandle, // Init DB on every generation.
	}, false
}

func (r *ApcSwitcher) Outputs() map[string]int {
	return map[string]int{}
}

func (r *ApcSwitcher) Config(params map[string]interface{}) {

}

func (r *ApcSwitcher) OnStart() {
}

func (r *ApcSwitcher) OnMessageArrived(msgs []*actor.Message) error {
	switch msgs[0].Name {
	case actor.MsgApcHandle:
		var na int
		intf.Router.Call("scheduler", "NotifyApchandler", msgs[0].Height, &na)
	}
	return nil
}
