package exec

import (
	"context"
	"sync"

	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/arcology-network/evm/common"
	"go.uber.org/zap"
)

type RpcService struct {
	actor.WorkerThread

	pendingTxs      map[uint64]chan []*ExecutorResponse
	pendingTxsGuard sync.Mutex
}

var (
	rpcInstance actor.IWorkerEx
	initRpcOnce sync.Once
)

func NewRpcService(concurrency int, groupId string) actor.IWorkerEx {
	initRpcOnce.Do(func() {
		rpc := &RpcService{
			pendingTxs: make(map[uint64]chan []*ExecutorResponse),
		}
		rpc.Set(concurrency, groupId)

		rpcInstance = rpc
	})
	return rpcInstance
}

func (rpc *RpcService) Inputs() ([]string, bool) {
	return []string{
		actor.MsgTxsExecuteResults,
	}, false
}

func (rpc *RpcService) Outputs() map[string]int {
	return map[string]int{
		actor.MsgTxsToExecute: 1,
	}
}

func (rpc *RpcService) OnStart() {}

func (rpc *RpcService) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	rpc.pendingTxsGuard.Lock()
	defer rpc.pendingTxsGuard.Unlock()

	if ch, ok := rpc.pendingTxs[msg.Msgid]; ok {
		ch <- msg.Data.([]*ExecutorResponse)
		delete(rpc.pendingTxs, msg.Msgid)
	} else {
		panic("unexpected msg got")
	}
	return nil
}

func (rpc *RpcService) GetConfig(ctx context.Context, _ *int, config *cmntyp.ExecutorConfig) error {
	config.Concurrency = rpc.Concurrency
	return nil
}

func (rpc *RpcService) ExecTxs(ctx context.Context, request *actor.Message, response *cmntyp.ExecutorResponses) error {
	chResults := make(chan []*ExecutorResponse)
	rpc.pendingTxsGuard.Lock()
	rpc.pendingTxs[request.Msgid] = chResults
	rpc.pendingTxsGuard.Unlock()

	rpc.MsgBroker.Send(actor.MsgTxsToExecute, request.Data, request.Height, request.Msgid)
	results := <-chResults

	// The following code were copied from exec v1.
	args := request.Data.(*cmntyp.ExecutorRequest)
	total := 0
	for _, sequence := range args.Sequences {
		total = total + len(sequence.Msgs)
	}
	// spawnedKeys := []evmCommon.Hash{}
	// spawnedHashes := []evmCommon.Hash{}
	resultLength := 0

	// relationKeys := make([]evmCommon.Hash, 0, len(args.Sequences))
	// relationSizes := make([]uint64, 0, len(args.Sequences))
	// relationValues := make([]evmCommon.Hash, 0, total)

	// DfCalls := make([]*cmntyp.DeferCall, 0, total)
	HashList := make([]evmCommon.Hash, 0, total)
	StatusList := make([]uint64, 0, total)
	GasUsedList := make([]uint64, 0, total)
	contractAddress := []evmCommon.Address{}

	// TxidsHash := make([]evmCommon.Hash, 0, total)
	// TxidsId := make([]uint32, 0, total)
	// TxidsAddress := make([]evmCommon.Address, 0, total)
	callResults := make([][]byte, 0, total)

	for _, exectorResponse := range results {

		contractAddress = append(contractAddress, exectorResponse.ContractAddress...)

		// for k, v := range exectorResponse.SpawnHashes {
		// 	spawnedKeys = append(spawnedKeys, k)
		// 	spawnedHashes = append(spawnedHashes, v)
		// }

		// for sequenceid, list := range exectorResponse.Relation {
		// 	relationKeys = append(relationKeys, sequenceid)
		// 	relationSizes = append(relationSizes, uint64(len(list)))
		// 	relationValues = append(relationValues, list...)
		// }
		resultLength = resultLength + len(exectorResponse.Responses)
		for _, txResponse := range exectorResponse.Responses {
			// if !args.Sequences[i].Parallel {
			// 	DfCalls = append(DfCalls, nil)
			// } else {
			// 	DfCalls = append(DfCalls, txResponse.DfCall)
			// }
			HashList = append(HashList, txResponse.Hash)
			StatusList = append(StatusList, txResponse.Status)
			GasUsedList = append(GasUsedList, txResponse.GasUsed)
		}

		// for hash, item := range exectorResponse.Txids {
		// 	TxidsHash = append(TxidsHash, hash)
		// 	TxidsId = append(TxidsId, item.Id)
		// 	TxidsAddress = append(TxidsAddress, item.ContractAddress)
		// }
		callResults = append(callResults, exectorResponse.CallResults...)
	}

	rpc.AddLog(log.LogLevel_Debug, "Exec return results***********", zap.Int("txResults", resultLength))

	// response.DfCalls = DfCalls
	response.HashList = HashList
	response.StatusList = StatusList
	response.GasUsedList = GasUsedList
	// response.SpawnedKeys = spawnedKeys
	// response.SpawnedTxs = spawnedHashes
	// response.RelationKeys = relationKeys
	// response.RelationSizes = relationSizes
	// response.RelationValues = relationValues
	response.ContractAddresses = contractAddress
	// response.TxidsHash = TxidsHash
	// response.TxidsId = TxidsId
	// response.TxidsAddress = TxidsAddress
	response.CallResults = callResults
	return nil
}
