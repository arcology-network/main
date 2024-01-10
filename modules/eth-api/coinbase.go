package ethapi

import (
	"fmt"

	"github.com/arcology-network/component-lib/actor"
)

type Coinbase struct {
	actor.WorkerThread
}

// return a Subscriber struct
func NewCoinbase(concurrency int, groupid string) actor.IWorkerEx {
	coinbase := Coinbase{}
	coinbase.Set(concurrency, groupid)
	return &coinbase
}

func (sq *Coinbase) Inputs() ([]string, bool) {
	return []string{
		actor.MsgCoinbase,
	}, false
}

func (sq *Coinbase) Outputs() map[string]int {
	return map[string]int{}
}

func (sq *Coinbase) Config(params map[string]interface{}) {

}

func (*Coinbase) OnStart() {

}

func (*Coinbase) Stop() {}

func (sq *Coinbase) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgCoinbase:
			coinbase := v.Data.(*actor.BlockStart)
			options.Coinbase = fmt.Sprintf("0x%x", coinbase.Coinbase.Bytes())
		}
	}
	return nil
}
