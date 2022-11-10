package scheduler

import (
	"math/big"
	"sync"
	"time"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
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
	execConfigs []*cmntyp.ExecutorConfig
}

func NewExecClient(executors []string, batchSize int) *ExecClient {
	execConfigs := make([]*cmntyp.ExecutorConfig, len(executors))
	for i, exec := range executors {
		var na int
		execConfigs[i] = &cmntyp.ExecutorConfig{}
		intf.Router.Call(exec, "GetConfig", &na, execConfigs[i])
	}

	return &ExecClient{
		executors:   executors,
		batchSize:   batchSize,
		execConfigs: execConfigs,
	}
}

func (client *ExecClient) Run(
	messages map[ethcmn.Hash]*schtyp.Message,
	sequences []*cmntyp.ExecutingSequence,
	timestamp *big.Int,
	msgTemplate *actor.Message,
	inlog *actor.WorkerThreadLogger,
	parallelism int,
	generationIdx,
	batchIdx int,
) (
	map[ethcmn.Hash]*cmntyp.ExecuteResponse,
	map[ethcmn.Hash]ethcmn.Hash,
	map[ethcmn.Hash][]ethcmn.Hash,
	[]ethcmn.Address,
	map[ethcmn.Hash]uint32,
	map[ethcmn.Hash]ethcmn.Address,
) {
	requests := make([]*cmntyp.ExecutorRequest, 0, int(50000/client.batchSize))
	for _, sequence := range sequences {
		if sequence.Parallel {
			msg := messages[sequence.Msgs[0].TxHash]
			for i := 0; i < len(sequence.Msgs); i += client.batchSize {
				end := cmncmn.Min(len(sequence.Msgs), i+client.batchSize)
				requests = append(requests, &cmntyp.ExecutorRequest{
					Sequences: []*cmntyp.ExecutingSequence{
						{
							Msgs:     sequence.Msgs[i:end],
							Txids:    sequence.Txids[i:end],
							Parallel: true,
						},
					},
					Precedings:    [][]*ethcmn.Hash{*msg.Precedings},
					PrecedingHash: []ethcmn.Hash{msg.PrecedingHash},
					Timestamp:     timestamp,
					Debug:         false,
				})
			}
		} else {
			msg := messages[sequence.SequenceId]
			requests = append(requests, &cmntyp.ExecutorRequest{
				Sequences:     []*cmntyp.ExecutingSequence{sequence},
				Precedings:    [][]*ethcmn.Hash{*msg.Precedings},
				PrecedingHash: []ethcmn.Hash{msg.PrecedingHash},
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
	responses := make([][]*cmntyp.ExecutorResponses, len(client.executors))
	inlog.CheckPoint("exec preparation complete,start exec transactions", zap.Int("parallelism", parallelism), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
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
		go func(requests []*cmntyp.ExecutorRequest, execIdx int, requestIdx int, msg actor.Message) {
			data := mergeRequests(requests)
			msg.Name = actor.MsgTxsToExecute
			msg.Msgid = cmncmn.GenerateUUID()
			msg.Data = data

			inlog.CheckPoint(">>>>>>>>>>>>>>>>>>>>>", zap.Int("execIdx", execIdx), zap.Int("idx", requestIdx), zap.Int("sequences", len(data.Sequences)), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
			response := cmntyp.ExecutorResponses{}
			err := intf.Router.Call(client.executors[execIdx], "ExecTxs", &msg, &response)
			if err != nil {
				panic(err)
			}
			inlog.CheckPoint("<<<<<<<<<<<<<<<<<<<<<", zap.Int("execIdx", execIdx), zap.Int("idx", requestIdx), zap.Int("sequences", len(data.Sequences)), zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
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
	inlog.CheckPoint(".......................................................... exec completed", zap.Int("generationIdx", generationIdx), zap.Int("batchIdx", batchIdx))
	ExecTime.Observe(time.Since(execBegin).Seconds())
	ExecTimeGauge.Set(time.Since(execBegin).Seconds())

	// The following code were copied from exec v1.
	results := make(map[ethcmn.Hash]*cmntyp.ExecuteResponse)
	spawnedTxs := make(map[ethcmn.Hash]ethcmn.Hash)
	relations := make(map[ethcmn.Hash][]ethcmn.Hash)
	contractAddress := make([]ethcmn.Address, 0, 10)
	txids := make(map[ethcmn.Hash]uint32)
	defCallees := make(map[ethcmn.Hash]ethcmn.Address)
	for _, resps := range responses {
		for _, r := range resps {
			if r.DfCalls != nil {
				for i := range r.DfCalls {
					results[r.HashList[i]] = &cmntyp.ExecuteResponse{
						DfCall:  r.DfCalls[i],
						Hash:    r.HashList[i],
						Status:  r.StatusList[i],
						GasUsed: r.GasUsedList[i],
					}
				}
			}

			for i, spawnedHash := range r.SpawnedTxs {
				spawnedTxs[r.SpawnedKeys[i]] = spawnedHash
			}

			if len(r.RelationKeys) == len(r.RelationSizes) {
				idx := 0
				for i, hash := range r.RelationKeys {
					relations[hash] = r.RelationValues[idx : idx+int(r.RelationSizes[i])]
					idx = idx + int(r.RelationSizes[i])
				}
			}

			contractAddress = append(contractAddress, r.ContractAddresses...)

			if len(r.TxidsHash) == len(r.TxidsId) {
				for i, hash := range r.TxidsHash {
					txids[hash] = r.TxidsId[i]
					defCallees[hash] = r.TxidsAddress[i]
				}
			}
		}
	}
	return results, spawnedTxs, relations, contractAddress, txids, defCallees
}

func mergeRequests(requests []*cmntyp.ExecutorRequest) *cmntyp.ExecutorRequest {
	sequences := make([]*cmntyp.ExecutingSequence, len(requests))
	precedings := make([][]*ethcmn.Hash, len(requests))
	precedingHash := make([]ethcmn.Hash, len(requests))
	for i, request := range requests {
		sequences[i] = request.Sequences[0]
		precedings[i] = request.Precedings[0]
		precedingHash[i] = request.PrecedingHash[0]
	}
	return &cmntyp.ExecutorRequest{
		Sequences:     sequences,
		Precedings:    precedings,
		PrecedingHash: precedingHash,
		Timestamp:     requests[0].Timestamp,
		Debug:         requests[0].Debug,
	}
}
