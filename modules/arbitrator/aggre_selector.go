package arbitrator

import (
	"time"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/aggregator/aggregator"
	"github.com/arcology-network/component-lib/log"
	arbitrator "github.com/arcology-network/main/modules/arbitrator/impl-arbitrator"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"go.uber.org/zap"
)

type EuResultsAggreSelector struct {
	actor.WorkerThread
	ag         *aggregator.Aggregator
	arbitrator *arbitrator.ArbitratorImpl
}

// return a Subscriber struct
func NewEuResultsAggreSelector(concurrency int, groupid string) actor.IWorkerEx {
	agg := EuResultsAggreSelector{}
	agg.Set(concurrency, groupid)
	agg.ag = aggregator.NewAggregator()
	agg.arbitrator = arbitrator.NewArbitratorImpl()
	return &agg
}

func (a *EuResultsAggreSelector) OnStart() {
}

func (a *EuResultsAggreSelector) Inputs() ([]string, bool) {
	return []string{
		actor.MsgBlockCompleted,
		actor.MsgArbitrateReapinglist,
		actor.MsgInclusive,
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
		a.arbitrator.Clear()
		types.ProcessedEuResultPool.ReclaimRecursive()
		a.AddLog(log.LogLevel_Info, "clear pool", zap.Int("remainingQuantity", remainingQuantity), zap.Duration("arbitrator engine clear time", time.Since(t)))
	case actor.MsgArbitrateReapinglist:
		reapinglist := msgs[0].Data.(*ctypes.ReapingList)
		result, _ := a.ag.OnListReceived(reapinglist)
		a.SendMsg(result)
	case actor.MsgInclusive:
		inclusive := msgs[0].Data.(*ctypes.InclusiveList)
		inclusive.Mode = ctypes.InclusiveMode_Results
		a.ag.OnClearListReceived(inclusive)
	case actor.MsgPreProcessedEuResults:
		data := msgs[0].Data.([]*types.ProcessedEuResult)
		tim, nums := a.arbitrator.Insert(data)
		a.AddLog(log.LogLevel_Debug, "insert accessRecord***********", zap.Int("counts", nums), zap.Durations("time", tim))

		if len(data) > 0 {
			for _, v := range data {
				euResult := v
				result := a.ag.OnDataReceived(ethCommon.BytesToHash([]byte(euResult.Hash)), euResult)
				a.SendMsg(result)
			}
		}
	}
	return nil
}
func (a *EuResultsAggreSelector) SendMsg(selectedData *[]*interface{}) {
	if selectedData != nil {
		euResults := make([]*types.ProcessedEuResult, len(*selectedData))
		for i, euResult := range *selectedData {
			euResults[i] = (*euResult).(*types.ProcessedEuResult)
		}
		a.AddLog(log.LogLevel_CheckPoint, "send gather result", zap.Int("counts", len(euResults)))
		a.MsgBroker.Send(actor.MsgEuResultSelected, &euResults)
	}
}
