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
	"math/big"
	"sync"
	"time"

	cmncmn "github.com/arcology-network/common-lib/common"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	evmCommon "github.com/ethereum/go-ethereum/common"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	ExecTime = prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
		Subsystem: "scheduler",
		Name:      "exec_seconds",
		Help:      "The duration of execution step.",
	}, []string{})
	ExecTimeGauge = prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
		Subsystem: "scheduler",
		Name:      "exec_seconds_gauge",
		Help:      "The duration of execution step.",
	}, []string{})
)

type ExecClient struct {
	executors   []string
	batchSize   int
	execConfigs []*mtypes.ExecutorConfig
}

func NewExecClient(executors []string, batchSize int) *ExecClient {
	execConfigs := make([]*mtypes.ExecutorConfig, len(executors))
	for i, exec := range executors {
		var na int
		execConfigs[i] = &mtypes.ExecutorConfig{}
		intf.Router.Call(exec, "GetConfig", &na, execConfigs[i])
	}

	return &ExecClient{
		executors:   executors,
		batchSize:   batchSize,
		execConfigs: execConfigs,
	}
}

func (client *ExecClient) Run(
	// messages map[evmCommon.Hash]*schtyp.Message,
	sequences []*mtypes.ExecutingSequence,
	timestamp *big.Int,
	msgTemplate *actor.Message,
	inlog *actor.WorkerThreadLogger,
	height uint64,
	parallelism int,
	generationIdx int,
) (
	map[evmCommon.Hash]*mtypes.ExecuteResponse,
	[]evmCommon.Address,
) {
	requests := make([]*mtypes.ExecutorRequest, 0, int(50000/client.batchSize))
	for _, sequence := range sequences {
		if sequence.Parallel {
			for i := 0; i < len(sequence.Msgs); i += client.batchSize {
				end := cmncmn.Min(len(sequence.Msgs), i+client.batchSize)
				requests = append(requests, &mtypes.ExecutorRequest{
					Sequences: []*mtypes.ExecutingSequence{
						{
							Msgs:     sequence.Msgs[i:end],
							Parallel: true,
							GroupIds: sequence.GroupIds[i:end],
						},
					},
					Height:        height,
					GenerationIdx: uint32(generationIdx),
					Timestamp:     timestamp,
					Debug:         false,
				})
			}
		} else {
			requests = append(requests, &mtypes.ExecutorRequest{
				Sequences:     []*mtypes.ExecutingSequence{sequence},
				Height:        height,
				GenerationIdx: uint32(generationIdx),
				Timestamp:     timestamp,
				Debug:         false,
			})
		}
	}

	idleExec := make(chan int, len(client.executors))
	for i := range client.executors {
		idleExec <- i
	}

	execBegin := time.Now()
	concurrency := 0
	var ccGuard sync.Mutex
	var wg sync.WaitGroup
	//requestIdx := 0
	responses := make([][]*mtypes.ExecutorResponses, len(client.executors))
	inlog.CheckPoint("exec preparation complete,start exec transactions", zap.Int("parallelism", parallelism), zap.Int("generationIdx", generationIdx))
	for i := 0; i < len(requests); {
		execIdx := <-idleExec
		// Reach concurrency limit.
		if concurrency >= parallelism {
			continue
		}

		numThread := cmncmn.Min(parallelism-concurrency, client.execConfigs[execIdx].Concurrency)
		numThread = cmncmn.Min(numThread, len(requests)-i)
		ccGuard.Lock()
		concurrency += numThread
		ccGuard.Unlock()

		wg.Add(1)
		go func(requests []*mtypes.ExecutorRequest, execIdx int, requestIdx int, msg actor.Message) {
			data := mergeRequests(requests)
			msg.Name = actor.MsgTxsToExecute
			msg.Msgid = cmncmn.GenerateUUID()
			msg.Data = data

			inlog.CheckPoint(">>>>>>>>>>>>>>>>>>>>>", zap.Int("execIdx", execIdx), zap.Int("idx", requestIdx), zap.Int("sequences", len(data.Sequences)), zap.Int("generationIdx", generationIdx))
			response := mtypes.ExecutorResponses{}
			err := intf.Router.Call(client.executors[execIdx], "ExecTxs", &msg, &response)
			if err != nil {
				panic(err)
			}
			inlog.CheckPoint("<<<<<<<<<<<<<<<<<<<<<", zap.Int("execIdx", execIdx), zap.Int("idx", requestIdx), zap.Int("sequences", len(data.Sequences)), zap.Int("generationIdx", generationIdx))
			responses[execIdx] = append(responses[execIdx], &response)
			ccGuard.Lock()
			concurrency -= len(requests)
			ccGuard.Unlock()
			idleExec <- execIdx
			wg.Done()
		}(requests[i:i+numThread], execIdx, i, *msgTemplate)
		i += numThread
	}
	wg.Wait()
	inlog.CheckPoint(".......................................................... exec completed", zap.Int("generationIdx", generationIdx))
	ExecTime.Observe(time.Since(execBegin).Seconds())
	ExecTimeGauge.Set(time.Since(execBegin).Seconds())

	// The following code were copied from exec v1.
	results := make(map[evmCommon.Hash]*mtypes.ExecuteResponse)

	contractAddress := make([]evmCommon.Address, 0, 10)

	for _, resps := range responses {
		for _, r := range resps {
			for i := range r.HashList {
				results[r.HashList[i]] = &mtypes.ExecuteResponse{
					Hash:    r.HashList[i],
					Status:  r.StatusList[i],
					GasUsed: r.GasUsedList[i],
				}
			}

			contractAddress = append(contractAddress, r.ContractAddresses...)

		}
	}
	return results, contractAddress
}

func mergeRequests(requests []*mtypes.ExecutorRequest) *mtypes.ExecutorRequest {
	sequences := make([]*mtypes.ExecutingSequence, len(requests))
	for i, request := range requests {
		sequences[i] = request.Sequences[0]
	}
	return &mtypes.ExecutorRequest{
		Sequences:     sequences,
		Height:        requests[0].Height,
		GenerationIdx: requests[0].GenerationIdx,
		Timestamp:     requests[0].Timestamp,
		Debug:         requests[0].Debug,
	}
}
