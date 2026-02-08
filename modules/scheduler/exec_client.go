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

	cmncmn "github.com/arcology-network/common-lib/common"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/logger"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
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
	batchSize int
	// parallelism int
	executors map[int]*mtypes.ExecutorConf
}

func NewExecClient(batchSize int, executors []*mtypes.ExecutorConf) *ExecClient {
	executor_dic := map[int]*mtypes.ExecutorConf{}
	for i := range executors {
		executor_dic[i] = executors[i]
	}
	return &ExecClient{
		batchSize: batchSize,
		executors: executor_dic,
	}
}

// type ExecutorResult struct {
// 	ExecIdx  int
// 	Response *mtypes.ExecutorResponses
// }

// func (client *ExecClient) ExecuteDynamically(
// 	ctx *actor.ExecutionContext,
// 	requests []*mtypes.ExecutorRequest,
// 	executors int,
// ) [][]*mtypes.ExecutorResponses {

// 	taskChan := make(chan *mtypes.ExecutorRequest)
// 	resultChan := make(chan ExecutorResult, len(requests))

// 	// 启动 executors 个 worker
// 	var wg sync.WaitGroup
// 	for execIdx := 0; execIdx < executors; execIdx++ {
// 		wg.Add(1)
// 		go func(execIdx int, ctx *actor.ExecutionContext) {
// 			defer wg.Done()

// 			cap := client.getParallelism(execIdx)
// 			batch := make([]*mtypes.ExecutorRequest, 0, cap)
// 			for {
// 				// 拉取一批任务
// 				for i := 0; i < cap; i++ {
// 					req, ok := <-taskChan
// 					if !ok {
// 						// 通道关闭，但如果 batch 还有任务，要执行完
// 						if len(batch) > 0 {
// 							goto EXEC
// 						}
// 						// 没有任务直接退出 worker
// 						return
// 					}
// 					batch = append(batch, req)
// 				}

// 			EXEC:
// 				if len(batch) == 0 {
// 					return
// 				}

// 				// 执行
// 				data := mergeRequests(batch)

// 				resp, err := ctx.SendSync("executor", "ExecTxs", data, data.Height)
// 				if err != nil {
// 					ctx.LogErr("request executor.ExecTxs error", logger.F("err", err.Error()))
// 					return
// 				}

// 				resultChan <- ExecutorResult{execIdx, resp.(*mtypes.ExecutorResponses)}
// 			}
// 		}(execIdx, ctx.Copy())
// 	}

// 	// 投递任务（放入统一任务池）
// 	go func() {
// 		for _, req := range requests {
// 			taskChan <- req
// 		}
// 		close(taskChan)
// 	}()

// 	// 收集结果
// 	responses := make([][]*mtypes.ExecutorResponses, executors)

// 	for i := 0; i < len(requests); i++ {
// 		res := <-resultChan
// 		responses[res.ExecIdx] = append(responses[res.ExecIdx], res.Response)
// 	}

// 	close(resultChan)
// 	wg.Wait()

// 	return responses
// }

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
		// Debug:         requests[0].Debug,
	}
}

func (client *ExecClient) StartIssue(
	exectx *actor.ExecutionContext,
	genCtx *generationContext,
) {
	for execId := range client.executors {
		if client.Issue(exectx, genCtx, execId) {
			return
		}
	}
}

func (client *ExecClient) Issue(
	exectx *actor.ExecutionContext,
	genCtx *generationContext,
	execId int,
) bool {
	finish := len(genCtx.remaining) == 0
	if finish {
		return finish
	}

	eins, ok := client.executors[execId]
	if !ok {
		exectx.LogErr("not found executor", logger.F("execId", execId))
		return false
	}

	cap := eins.Eus
	reqs, finish := genCtx.execIssue(execId, cap)

	request := mergeRequests(reqs)
	request.ExecId = uint32(execId)
	exectx.InvokeRPC(eins.Name, "startExecute", request, "onExecResult")
	return finish
}

func (client *ExecClient) buildExecutorRequests(sequences []*mtypes.ExecutingSequence, timestamp *big.Int, height uint64, generationIdx int) []*mtypes.ExecutorRequest {
	requests := make([]*mtypes.ExecutorRequest, 0, int(maxBlockSize/client.batchSize))
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
					// Debug:         false,
				})
			}
		} else {
			requests = append(requests, &mtypes.ExecutorRequest{
				Sequences:     []*mtypes.ExecutingSequence{sequence},
				Height:        height,
				GenerationIdx: uint32(generationIdx),
				Timestamp:     timestamp,
				// Debug:         false,
			})
		}

	}
	return requests
}

// func (client *ExecClient) Run(
// 	sequences []*mtypes.ExecutingSequence,
// 	timestamp *big.Int,
// 	ctx *actor.ExecutionContext,
// 	height uint64,
// 	executors int,
// 	parallelism int,
// 	generationIdx int,
// ) (
// 	map[evmCommon.Hash]*mtypes.ExecuteResponse,
// 	[]evmCommon.Address,
// ) {

// 	execBegin := time.Now()
// 	responses := client.ExecuteDynamically(ctx, requests, executors)

// 	logger.Log.Info(context.Background(), "exec completed", logger.F("generationIdx", generationIdx))

// 	ExecTime.Observe(time.Since(execBegin).Seconds())
// 	ExecTimeGauge.Set(time.Since(execBegin).Seconds())

// 	// The following code were copied from exec v1.
// 	results := make(map[evmCommon.Hash]*mtypes.ExecuteResponse)

// 	contractAddress := make([]evmCommon.Address, 0, 10)

// 	for _, resps := range responses {
// 		for _, r := range resps {
// 			for i := range r.HashList {
// 				results[r.HashList[i]] = &mtypes.ExecuteResponse{
// 					Hash:    r.HashList[i],
// 					Status:  r.StatusList[i],
// 					GasUsed: r.GasUsedList[i],
// 				}
// 			}

// 			contractAddress = append(contractAddress, r.ContractAddresses...)

// 		}
// 	}
// 	return results, contractAddress
// }
