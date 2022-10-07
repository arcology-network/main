package scheduler

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/log"
	schedulingTypes "github.com/HPISTechnologies/main/modules/scheduler/types"
	prometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type RpcClientExec struct {
	rpcClients []string
	batchNums  int
}

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
	execBegin time.Time
)

//return a Subscriber struct
func NewRpcClientExec(execAddrs []string, batchNums int) *RpcClientExec {
	return &RpcClientExec{
		batchNums:  batchNums,
		rpcClients: execAddrs,
	}
}

func (rce *RpcClientExec) Start() {}

func (rce *RpcClientExec) Stop() {}

func makeExecutorRequest(hash ethCommon.Hash, precedings []*ethCommon.Hash, sequence *types.ExecutingSequence, timestamp *big.Int, parallelism int) *types.ExecutorRequest {
	return &types.ExecutorRequest{
		Sequences:     []*types.ExecutingSequence{sequence},
		Precedings:    precedings,
		PrecedingHash: hash,
		Timestamp:     timestamp,
		Parallelism:   uint64(parallelism),
		Debug:         false,
	}
}

func mergeExecutorRequest(sequence []*types.ExecutorRequest) *types.ExecutorRequest {
	if len(sequence) == 0 {
		return &types.ExecutorRequest{}
	}
	finalSequences := make([]*types.ExecutingSequence, 0, len(sequence))
	for _, param := range sequence {
		finalSequences = append(finalSequences, param.Sequences...)
	}
	return &types.ExecutorRequest{
		Sequences:     finalSequences,
		Precedings:    sequence[0].Precedings,
		PrecedingHash: sequence[0].PrecedingHash,
		Timestamp:     sequence[0].Timestamp,
		Parallelism:   sequence[0].Parallelism,
	}
}

func (rce *RpcClientExec) Run(messages map[ethCommon.Hash]*schedulingTypes.Message, sequences []*types.ExecutingSequence, timestamp *big.Int, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, parallelism int, generationIdx, batchIdx int) (map[ethCommon.Hash]*types.ExecuteResponse, map[ethCommon.Hash]ethCommon.Hash, map[ethCommon.Hash][]ethCommon.Hash, []ethCommon.Address, map[ethCommon.Hash]uint32, map[ethCommon.Hash]ethCommon.Address) {
	startTime := time.Now()

	execParams := make([]*types.ExecutorRequest, 0, len(sequences))
	for _, sequence := range sequences {
		inlog.Log(log.LogLevel_Debug, "rpc_client_exec ", zap.Bool("Parallel", sequence.Parallel), zap.Int("msgs nums", len(sequence.Msgs)))

		precedingHash := ethCommon.Hash{}
		var precedings []*ethCommon.Hash

		if sequence.Parallel {
			if msg, ok := messages[sequence.Msgs[0].TxHash]; ok {
				precedingHash = msg.PrecedingHash
				precedings = *msg.Precedings
			} else {
				continue
			}

			msgs := []*types.StandardMessage{}
			txids := []uint32{}
			for i, msg := range sequence.Msgs {
				if len(msgs) >= rce.batchNums {
					currentSequence := &types.ExecutingSequence{
						Msgs:     msgs,
						Txids:    txids,
						Parallel: true,
					}
					execParam := makeExecutorRequest(precedingHash, precedings, currentSequence, timestamp, parallelism)
					execParams = append(execParams, execParam)
					msgs = []*types.StandardMessage{}
					txids = []uint32{}
				}
				msgs = append(msgs, msg)
				txids = append(txids, sequence.Txids[i])
			}
			if len(msgs) > 0 {
				lastSequence := &types.ExecutingSequence{
					Msgs:     msgs,
					Txids:    txids,
					Parallel: true,
				}
				execParam := makeExecutorRequest(precedingHash, precedings, lastSequence, timestamp, parallelism)
				execParams = append(execParams, execParam)
			}
		} else {
			if msg, ok := messages[sequence.SequenceId]; ok {
				precedingHash = msg.PrecedingHash
				precedings = *msg.Precedings
			} else {
				continue
			}

			execParam := makeExecutorRequest(precedingHash, precedings, sequence, timestamp, parallelism)
			execParams = append(execParams, execParam)
		}
	}
	inlog.Log(log.LogLevel_Debug, "execParams ", zap.Int("execParams nums", len(execParams)))
	params := map[ethCommon.Hash]*[]*types.ExecutorRequest{}
	for _, param := range execParams {
		if list, ok := params[param.PrecedingHash]; ok {
			(*list) = append((*list), param)
		} else {
			params[param.PrecedingHash] = &[]*types.ExecutorRequest{param}
		}
	}
	finalExecParams := make([]*types.ExecutorRequest, 0, len(execParams))
	for _, list := range params {
		executorRequests := []*types.ExecutorRequest{}
		for _, param := range *list {
			if len(executorRequests) >= parallelism {
				execParam := mergeExecutorRequest(executorRequests)
				finalExecParams = append(finalExecParams, execParam)
				executorRequests = []*types.ExecutorRequest{}
			}
			executorRequests = append(executorRequests, param)
		}
		if len(executorRequests) > 0 {
			execParam := mergeExecutorRequest(executorRequests)
			finalExecParams = append(finalExecParams, execParam)
		}
	}
	inlog.Log(log.LogLevel_Debug, "exec preparation complete,start exec transactions", zap.Int("parallelism", parallelism), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx), zap.Int("finalExecParams nums", len(finalExecParams)), zap.Duration("time", time.Since(startTime)))
	execBegin = time.Now()
	return rce.execTxs(&finalExecParams, msgTemplate, inlog, generationIdx, batchIdx)
}

func (rce *RpcClientExec) execTxs(params *[]*types.ExecutorRequest, msgTemplate *actor.Message, inlog *actor.WorkerThreadLogger, generationIdx, batchIdx int) (map[ethCommon.Hash]*types.ExecuteResponse, map[ethCommon.Hash]ethCommon.Hash, map[ethCommon.Hash][]ethCommon.Hash, []ethCommon.Address, map[ethCommon.Hash]uint32, map[ethCommon.Hash]ethCommon.Address) {
	chExes := make(chan int, len(rce.rpcClients))
	wg := sync.WaitGroup{}
	for tokenIdx := range rce.rpcClients {
		chExes <- tokenIdx
	}
	totalLen := 0
	execResults := make([][]*types.ExecuteResponse, len(*params))
	spawnedTxs := map[ethCommon.Hash]ethCommon.Hash{}
	relations := map[ethCommon.Hash][]ethCommon.Hash{}
	txids := map[ethCommon.Hash]uint32{}
	defCallees := map[ethCommon.Hash]ethCommon.Address{}
	contractAddress := []ethCommon.Address{}
	for dataIdx, param := range *params {
		for _, sequence := range param.Sequences {
			totalLen = totalLen + len(sequence.Msgs)
		}
		execParam := param
		execIdx := <-chExes
		wg.Add(1)
		go func(execIdx, dataIdx int, request actor.Message) {
			response := types.ExecutorResponses{}
			request.Name = actor.MsgTxsToExecute
			request.Msgid = common.GenerateUUID()
			request.Data = execParam

			inlog.Log(log.LogLevel_Debug, ">>>>>>>>>>>>>>>>>>>>>", zap.Int("execIdx", execIdx), zap.Int("idx", dataIdx), zap.Int("sequences", len(execParam.Sequences)), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
			err := intf.Router.Call(rce.rpcClients[execIdx], "ExecTxs", &request, &response)
			if err != nil {
				inlog.Log(log.LogLevel_Error, "exec err", zap.String("err", fmt.Sprintf("%v", err.Error())))
				return
			} else {

				if response.DfCalls != nil {
					defs := 0
					for i := range response.DfCalls {
						txResult := &types.ExecuteResponse{
							DfCall:  response.DfCalls[i],
							Hash:    response.HashList[i],
							Status:  response.StatusList[i],
							GasUsed: response.GasUsedList[i],
						}
						if txResult.DfCall != nil {
							defs = defs + 1
						}
						execResults[dataIdx] = append(execResults[dataIdx], txResult)
					}
					inlog.Log(log.LogLevel_Debug, "<<<<<<<<<<<<<<<<<<<<< ", zap.Int("idx", dataIdx), zap.Int("defs", defs), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
				}
				for i, spawnedHash := range response.SpawnedTxs {
					spawnedTxs[response.SpawnedKeys[i]] = spawnedHash
				}
				if len(response.RelationKeys) == len(response.RelationSizes) {
					idx := 0
					for i, hash := range response.RelationKeys {
						relations[hash] = response.RelationValues[idx : idx+int(response.RelationSizes[i])]
						idx = idx + int(response.RelationSizes[i])
					}
				}

				if len(response.TxidsHash) == len(response.TxidsId) {
					for i, hash := range response.TxidsHash {
						txids[hash] = response.TxidsId[i]
						defCallees[hash] = response.TxidsAddress[i]
					}
				}

				contractAddress = append(contractAddress, response.ContractAddresses...)
			}
			chExes <- execIdx
			wg.Done()
		}(execIdx, dataIdx, *msgTemplate)
	}
	wg.Wait()
	inlog.Log(log.LogLevel_Debug, ".......................................................... exec completed", zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
	ExecTime.Observe(time.Since(execBegin).Seconds())
	ExecTimeGauge.Set(time.Since(execBegin).Seconds())
	reponses := make(map[ethCommon.Hash]*types.ExecuteResponse, totalLen)
	for _, list := range execResults {
		for _, response := range list {
			reponses[response.Hash] = response
		}
	}
	return reponses, spawnedTxs, relations, contractAddress, txids, defCallees
}
