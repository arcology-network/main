package scheduler

import (
	"fmt"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/log"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	ArbTime = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Subsystem: "scheduler",
		Name:      "arb_seconds",
		Help:      "The duration of arbitration step.",
	}, []string{})
	ArbTimeGauge = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Subsystem: "scheduler",
		Name:      "arb_seconds_gauge",
		Help:      "The duration of arbitration step.",
	}, []string{})
	arbBegin time.Time
)

type RpcClientArbitrate struct{}

func NewRpcClientArbitrate() *RpcClientArbitrate {
	return &RpcClientArbitrate{}
}

func (rca *RpcClientArbitrate) Start() {

}

func (rca *RpcClientArbitrate) Stop() {

}

func (rca *RpcClientArbitrate) Do(arbitrateList [][][]*types.TxElement, inlog *actor.WorkerThreadLogger, generationIdx, batchIdx int) ([]*ethCommon.Hash, []uint32, []uint32) {
	results := make([]*ethCommon.Hash, 0, len(arbitrateList))
	cpairLeft := make([]uint32, 0, len(arbitrateList))
	cpairRight := make([]uint32, 0, len(arbitrateList))
	for i, list := range arbitrateList {
		if len(list) == 0 {
			continue
		}
		request := actor.Message{
			Msgid: common.GenerateUUID(),
			Name:  actor.MsgArbitrateList,
			Data: &types.ArbitratorRequest{
				TxsListGroup: list,
			},
			Height: inlog.LatestMessage.Height,
			Round:  inlog.LatestMessage.Round,
		}
		response := types.ArbitratorResponse{}

		inlog.Log(log.LogLevel_Debug, "start arbitrate >>>>>>>>>>>>>>>>>>>", zap.Int("group idx", i), zap.Int("txs", len(list)), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
		arbBegin = time.Now()
		err := intf.Router.Call("arbitrator", "Arbitrate", &request, &response)
		if err != nil {
			inlog.Log(log.LogLevel_Error, "arbitrate err", zap.String("err", fmt.Sprintf("%v", err.Error())))
			return nil, nil, nil
		} else {
			inlog.Log(log.LogLevel_Debug, "return arbitrate <<<<<<<<<<<<<<<<<<<<", zap.Int("group idx", i), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
			ArbTime.Observe(time.Since(arbBegin).Seconds())
			ArbTimeGauge.Set(time.Since(arbBegin).Seconds())
			if response.ConflictedList != nil {
				results = append(results, response.ConflictedList...)
			}
			if response.CPairLeft != nil {
				cpairLeft = response.CPairLeft
			}
			if response.CPairRight != nil {
				cpairRight = response.CPairRight
			}
		}
	}
	return results, cpairLeft, cpairRight
}
