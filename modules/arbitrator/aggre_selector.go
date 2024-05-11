package arbitrator

import (
	"time"

	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/aggregator/aggregator"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type EuResultsAggreSelector struct {
	actor.WorkerThread
	ag *aggregator.Aggregator
	// arbitrator *arbitrator.ArbitratorImpl
}

// return a Subscriber struct
func NewEuResultsAggreSelector(concurrency int, groupid string) actor.IWorkerEx {
	agg := EuResultsAggreSelector{}
	agg.Set(concurrency, groupid)
	agg.ag = aggregator.NewAggregator()
	// agg.arbitrator = arbitrator.NewArbitratorImpl()
	return &agg
}

func (a *EuResultsAggreSelector) OnStart() {
}

func (a *EuResultsAggreSelector) Inputs() ([]string, bool) {
	return []string{
		actor.MsgBlockCompleted,
		actor.MsgArbitrateReapinglist,
		actor.MsgPreProcessedEuResults,
	}, false
}

func (a *EuResultsAggreSelector) Outputs() map[string]int {
	return map[string]int{
		actor.MsgEuResultSelected: 1,
	}
}

func (a *EuResultsAggreSelector) OnMessageArrived(msgs []*actor.Message) error {
	switch msgs[0].Name {
	case actor.MsgBlockCompleted:
		remainingQuantity := a.ag.OnClearInfoReceived()
		t := time.Now()
		// a.arbitrator.Clear()
		types.RecordPool.ReclaimRecursive()
		a.AddLog(log.LogLevel_Info, "clear pool", zap.Int("remainingQuantity", remainingQuantity), zap.Duration("arbitrator engine clear time", time.Since(t)))
	case actor.MsgArbitrateReapinglist:
		reapinglist := msgs[0].Data.(*ctypes.ReapingList)
		result, _ := a.ag.OnListReceived(reapinglist)
		a.SendMsg(result)

	case actor.MsgPreProcessedEuResults:
		data := msgs[0].Data.([]*types.AccessRecord)

		if len(data) > 0 {
			for _, v := range data {
				euResult := v
				result := a.ag.OnDataReceived(evmCommon.BytesToHash(euResult.TxHash[:]), euResult)
				a.SendMsg(result)
			}
		}
	}
	return nil
}
func (a *EuResultsAggreSelector) SendMsg(selectedData *[]*interface{}) {
	if selectedData != nil {
		euResults := make([]*types.AccessRecord, len(*selectedData))
		for i, euResult := range *selectedData {
			euResults[i] = (*euResult).(*types.AccessRecord)
		}
		a.AddLog(log.LogLevel_CheckPoint, "send gather result", zap.Int("counts", len(euResults)))
		a.MsgBroker.Send(actor.MsgEuResultSelected, &euResults)
	}
}
