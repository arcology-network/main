package arbitrator

import (
	"github.com/arcology-network/common-lib/common"
	eushared "github.com/arcology-network/eu/shared"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"github.com/arcology-network/streamer/actor"
)

type EuResultPreProcessor struct {
	actor.WorkerThread
}

func NewEuResultPreProcessor(concurrency int, groupid string) actor.IWorkerEx {
	p := &EuResultPreProcessor{}
	p.Set(concurrency, groupid)
	return p
}

func (p *EuResultPreProcessor) OnStart() {

}

func (p *EuResultPreProcessor) Inputs() ([]string, bool) {
	return []string{actor.MsgTxAccessRecords}, false
}

func (p *EuResultPreProcessor) Outputs() map[string]int {
	return map[string]int{
		actor.MsgPreProcessedEuResults: 100,
	}
}

func (p *EuResultPreProcessor) OnMessageArrived(msgs []*actor.Message) error {
	results := *(msgs[0].Data.(*eushared.TxAccessRecordSet))

	processed := make([]*types.AccessRecord, len(results))
	worker := func(start, end, idx int, args ...interface{}) {
		recordPool := types.RecordPool
		uniPool := types.UnivaluePool
		for i := start; i < end; i++ {
			processed[i] = types.Decode(results[i], recordPool, uniPool)
		}
	}
	common.ParallelWorker(len(results), p.Concurrency, worker)

	p.MsgBroker.Send(actor.MsgPreProcessedEuResults, processed)
	return nil
}
