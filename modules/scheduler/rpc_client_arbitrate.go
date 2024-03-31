package scheduler

import (
	"fmt"
	"time"

	"github.com/arcology-network/common-lib/common"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
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

func (rca *RpcClientArbitrate) Do(arbitrateList [][]evmCommon.Hash, inlog *actor.WorkerThreadLogger, generationIdx int) ([]evmCommon.Hash, []uint32, []uint32) {
	results := make([]evmCommon.Hash, 0, len(arbitrateList))
	cpairLeft := make([]uint32, 0, len(arbitrateList))
	cpairRight := make([]uint32, 0, len(arbitrateList))

	request := actor.Message{
		Msgid: common.GenerateUUID(),
		Name:  actor.MsgArbitrateList,
		Data: &mtypes.ArbitratorRequest{
			TxsListGroup: arbitrateList,
		},
		Height: inlog.LatestMessage.Height,
		Round:  inlog.LatestMessage.Round,
	}
	response := mtypes.ArbitratorResponse{}

	inlog.CheckPoint("start arbitrate >>>>>>>>>>>>>>>>>>>", zap.Int("txs", len(arbitrateList)), zap.Int("generationIdx", generationIdx))
	arbBegin = time.Now()
	err := intf.Router.Call("arbitrator", "Arbitrate", &request, &response)
	if err != nil {
		inlog.Log(log.LogLevel_Error, "arbitrate err", zap.String("err", fmt.Sprintf("%v", err.Error())))
		return nil, nil, nil
	} else {
		inlog.CheckPoint("return arbitrate <<<<<<<<<<<<<<<<<<<<", zap.Int("generationIdx", generationIdx))
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
	return results, cpairLeft, cpairRight
}
