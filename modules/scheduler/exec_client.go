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

	mtypes "github.com/arcology-network/main/types"
	workload "github.com/arcology-network/scheduler/workload"
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

func MakeRequests(requests []*workload.JobSequence, timestamp *big.Int, height uint64, genidx int) *mtypes.ExecutorRequest {
	return &mtypes.ExecutorRequest{
		JobSequences:  requests,
		Height:        height,
		GenerationIdx: uint32(genidx),
		Timestamp:     timestamp,
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
	reqs, finish := genCtx.execIssue(execId, cap, client.batchSize)

	request := MakeRequests(reqs, genCtx.generation.context.timestamp, genCtx.generation.context.height, genCtx.genID)
	request.ExecId = uint32(execId)
	exectx.InvokeRPC(eins.Name, "startExecute", request, "onExecResult")
	return finish
}
