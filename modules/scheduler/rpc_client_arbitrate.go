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
	"time"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
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

// func (rca *RpcClientArbitrate) Do(ctx *actor.ExecutionContext, arbitrateList [][]evmCommon.Hash, generationIdx int) ([]uint64, []uint64) {
// 	cpairLeft := make([]uint64, 0, len(arbitrateList))
// 	cpairRight := make([]uint64, 0, len(arbitrateList))

// 	ctx.LogInfo("start arbitrate", logger.F("txs", len(arbitrateList)), logger.F("generationIdx", generationIdx))
// 	arbBegin = time.Now()

// 	resp, err := ctx.SendSync("arbitrator", "Arbitrate", &mtypes.ArbitratorRequest{
// 		TxsListGroup: arbitrateList,
// 	})
// 	if err != nil {
// 		ctx.LogErr("arbitrate err", logger.F("err", err))
// 		return nil, nil
// 	} else {
// 		ctx.LogInfo("return arbitrate", logger.F("generationIdx", generationIdx))
// 		ArbTime.Observe(time.Since(arbBegin).Seconds())
// 		ArbTimeGauge.Set(time.Since(arbBegin).Seconds())
// 		response := resp.(*mtypes.ArbitratorResponse)
// 		if response.CPairLeft != nil {
// 			cpairLeft = response.CPairLeft
// 		}
// 		if response.CPairRight != nil {
// 			cpairRight = response.CPairRight
// 		}
// 	}
// 	return cpairLeft, cpairRight
// }

// ----------------------------------
func (rca *RpcClientArbitrate) Issue(
	ctx *actor.ExecutionContext,
	arbitrateList [][]evmCommon.Hash,
) {
	ctx.InvokeRPC("arbitrator", "startArbitrate", &mtypes.ArbitratorRequest{
		TxsListGroup: arbitrateList,
	}, "onArbResult")
}
