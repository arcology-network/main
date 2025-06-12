/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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

func (rca *RpcClientArbitrate) Do(arbitrateList [][]evmCommon.Hash, inlog *actor.WorkerThreadLogger, generationIdx int) ([]uint64, []uint64) {
	// results := make([]evmCommon.Hash, 0, len(arbitrateList))
	cpairLeft := make([]uint64, 0, len(arbitrateList))
	cpairRight := make([]uint64, 0, len(arbitrateList))

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
		return nil, nil
	} else {
		inlog.CheckPoint("return arbitrate <<<<<<<<<<<<<<<<<<<<", zap.Int("generationIdx", generationIdx))
		ArbTime.Observe(time.Since(arbBegin).Seconds())
		ArbTimeGauge.Set(time.Since(arbBegin).Seconds())

		if response.CPairLeft != nil {
			cpairLeft = response.CPairLeft
		}
		if response.CPairRight != nil {
			cpairRight = response.CPairRight
		}
	}
	return cpairLeft, cpairRight
}
