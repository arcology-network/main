package exec

import (
	"context"
	"math"
	"sync"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	kafkalib "github.com/HPISTechnologies/component-lib/kafka/lib"
	"github.com/HPISTechnologies/component-lib/log"
	execTypes "github.com/HPISTechnologies/main/modules/exec/types"
	"go.uber.org/zap"
)

type ExecutorResponse struct {
	Responses       []*types.ExecuteResponse
	Uuid            uint64
	SerialID        int
	Total           int
	Relation        map[ethCommon.Hash][]ethCommon.Hash
	SpawnHashes     map[ethCommon.Hash]ethCommon.Hash
	ContractAddress []ethCommon.Address
	Txids           map[ethCommon.Hash]*execTypes.DefItem
	CallResults     [][]byte
}
type PackMessage struct {
	msg      *actor.Message
	response chan []*ExecutorResponse
}
type RpcService struct {
	actor.WorkerThread
	wbs       *kafkalib.Waitobjs
	msgid     int64
	responses map[uint64][]*ExecutorResponse

	// msgsChan chan *actor.Message
	// exitChan chan bool
	mqtxs  *MessageQueue
	mqcmds *MessageQueue

	exitflag bool
}

var (
	rpcServiceSingleton actor.IWorkerEx
	initOnce            sync.Once
)

//return a Subscriber struct
func NewRpcService(concurrency int, groupid string) actor.IWorkerEx {
	initOnce.Do(
		func() {
			rs := RpcService{}
			rs.Set(concurrency, groupid)
			rs.msgid = 0
			rs.responses = map[uint64][]*ExecutorResponse{}
			// rs.msgsChan = make(chan *actor.Message, 50)
			// rs.exitChan = make(chan bool, 0)
			rpcServiceSingleton = &rs
			rs.mqtxs = NewMessageQueue()
			rs.mqcmds = NewMessageQueue()
			rs.exitflag = false
		},
	)
	return rpcServiceSingleton
}

func (rpc *RpcService) Inputs() ([]string, bool) {
	return []string{actor.MsgTxsExecuteResults, actor.MsgSelectedExecuted, actor.MsgApcHandle}, false
}

func (rpc *RpcService) Outputs() map[string]int {
	return map[string]int{actor.MsgTxs: 1, actor.MsgExecuted: 1}
}

func (rs *RpcService) execMsgs(msgpacks *PackMessage) {
	if msgpacks == nil {
		return
	}
	rs.msgid = rs.msgid + 1

	rs.wbs.AddWaiter(rs.msgid)

	rs.ChangeMessageHeight(msgpacks.msg.Height)

	args := msgpacks.msg.Data.(*types.ExecutorRequest)

	rs.MsgBroker.Send(actor.MsgTxs, args)
	rs.wbs.Waitforever(rs.msgid)

	results := rs.wbs.GetData(rs.msgid)

	var txResults []*ExecutorResponse
	if results != nil {
		if bValue, ok := results.([]*ExecutorResponse); ok {
			txResults = bValue
		} else {
			rs.AddLog(log.LogLevel_Error, "data type not mismatch error")
		}

	} else {
		rs.AddLog(log.LogLevel_Error, "data  error")
	}

	msgpacks.response <- txResults
}
func (rs *RpcService) OnStart() {
	rs.wbs = kafkalib.StartWaitObjects()
	go func() {
		ApcHeight := uint64(0)
		rpcTxsHeight := uint64(math.MaxUint64)
		canExecute := false
		for {
			if rs.exitflag {
				return
			}
			cmd := rs.mqcmds.GetNext()
			if cmd != nil {
				msg := cmd.(*actor.Message)
				switch msg.Name {
				case actor.MsgApcHandle:
					ApcHeight = msg.Height + 1
					rs.ChangeMessageHeight(ApcHeight)
					canExecute = true
				case actor.MsgSelectedExecuted:
					canExecute = false
					rs.MsgBroker.Send(actor.MsgExecuted, msg.Data, msg.Height)
				}
			}
			for {
				tx := rs.mqtxs.GetNext()
				if tx != nil {
					pack := tx.(*PackMessage)
					if canExecute {
						if pack.msg.Height == rpcTxsHeight {
							pack.msg.Height = ApcHeight
							rs.AddLog(log.LogLevel_Debug, "*********debug mode", zap.Uint64("msgpack.msg.Height", pack.msg.Height))
							rs.execMsgs(pack)
						} else if pack.msg.Height == ApcHeight {
							rs.execMsgs(pack)
							rs.AddLog(log.LogLevel_Debug, "*********rpc txs direct exec", zap.Uint64("ApcHeight", ApcHeight), zap.Uint64("msgpack.msg.Height", pack.msg.Height))
						} else {
							rs.mqtxs.Insert(pack)
							rs.AddLog(log.LogLevel_Debug, "*********rpc txs cached", zap.Uint64("ApcHeight", ApcHeight), zap.Uint64("msgpack.msg.Height", pack.msg.Height))
							break
						}
					} else {
						rs.mqtxs.Insert(pack)
						break
					}
				} else {
					break
				}
			}

		}
	}()
}

func (rs *RpcService) Stop() {
	rs.exitflag = true
}

func (rs *RpcService) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgTxsExecuteResults:
			exeResults := v.Data.(*ExecutorResponse)
			if exeResults != nil {
				list, ok := rs.responses[exeResults.Uuid]
				if !ok {
					list = make([]*ExecutorResponse, exeResults.Total)
					rs.responses[exeResults.Uuid] = list
				}
				list[exeResults.SerialID] = exeResults
				rs.AddLog(log.LogLevel_Debug, "received exec results*****subpack******", zap.Int("nums", len(exeResults.Responses)))
				for _, v := range list {
					if v == nil {
						return nil
					}
				}

				delete(rs.responses, exeResults.Uuid)
				rs.AddLog(log.LogLevel_Debug, "received exec results***********", zap.Int64("msgid", rs.msgid))
				rs.wbs.Update(rs.msgid, list)
			}
		case actor.MsgSelectedExecuted:
			rs.mqcmds.Append(v)
		case actor.MsgApcHandle:
			rs.mqcmds.Append(v)
		}
	}

	return nil
}
func (rs *RpcService) ChangeMessageHeight(height uint64) {
	rs.WorkerThread.LatestMessage.Height = height
	rs.WorkerThread.MsgBroker.LatestMessage.Height = height
}
func (rs *RpcService) ExecTxs(ctx context.Context, request *actor.Message, response *types.ExecutorResponses) error {

	pack := PackMessage{
		msg:      request,
		response: make(chan []*ExecutorResponse, 1),
	}

	args := request.Data.(*types.ExecutorRequest)
	total := 0
	for _, sequence := range args.Sequences {
		total = total + len(sequence.Msgs)
	}
	rs.ChangeMessageHeight(request.Height)
	rs.AddLog(log.LogLevel_Debug, "receieved exec request fron rpc***********", zap.Uint64("packheight", request.Height), zap.Int("sequence", len(args.Sequences)), zap.Int("request total txs", total))

	//rs.msgsChan <- &pack
	rs.mqtxs.Append(&pack)

	txResults := <-pack.response

	if txResults == nil {
		return nil
	}
	spawnedKeys := []ethCommon.Hash{}
	spawnedHashes := []ethCommon.Hash{}
	resultLength := 0

	relationKeys := make([]ethCommon.Hash, 0, len(args.Sequences))
	relationSizes := make([]uint64, 0, len(args.Sequences))
	relationValues := make([]ethCommon.Hash, 0, total)

	DfCalls := make([]*types.DeferCall, 0, total)
	HashList := make([]ethCommon.Hash, 0, total)
	StatusList := make([]uint64, 0, total)
	GasUsedList := make([]uint64, 0, total)
	contractAddress := []ethCommon.Address{}

	TxidsHash := make([]ethCommon.Hash, 0, total)
	TxidsId := make([]uint32, 0, total)
	TxidsAddress := make([]ethCommon.Address, 0, total)
	callResults := make([][]byte, 0, total)

	for i, exectorResponse := range txResults {

		contractAddress = append(contractAddress, exectorResponse.ContractAddress...)

		for k, v := range exectorResponse.SpawnHashes {
			spawnedKeys = append(spawnedKeys, k)
			spawnedHashes = append(spawnedHashes, v)
		}

		for sequenceid, list := range exectorResponse.Relation {
			relationKeys = append(relationKeys, sequenceid)
			relationSizes = append(relationSizes, uint64(len(list)))
			relationValues = append(relationValues, list...)
		}
		resultLength = resultLength + len(exectorResponse.Responses)
		for _, txResponse := range exectorResponse.Responses {
			if !args.Sequences[i].Parallel {
				DfCalls = append(DfCalls, nil)
			} else {
				DfCalls = append(DfCalls, txResponse.DfCall)
			}
			HashList = append(HashList, txResponse.Hash)
			StatusList = append(StatusList, txResponse.Status)
			GasUsedList = append(GasUsedList, txResponse.GasUsed)
		}

		for hash, item := range exectorResponse.Txids {
			TxidsHash = append(TxidsHash, hash)
			TxidsId = append(TxidsId, item.Id)
			TxidsAddress = append(TxidsAddress, item.ContractAddress)
		}
		callResults = append(callResults, exectorResponse.CallResults...)
	}

	rs.AddLog(log.LogLevel_Debug, "Exec return results***********", zap.Int("txResults", resultLength))

	response.DfCalls = DfCalls
	response.HashList = HashList
	response.StatusList = StatusList
	response.GasUsedList = GasUsedList
	response.SpawnedKeys = spawnedKeys
	response.SpawnedTxs = spawnedHashes
	response.RelationKeys = relationKeys
	response.RelationSizes = relationSizes
	response.RelationValues = relationValues
	response.ContractAddresses = contractAddress
	response.TxidsHash = TxidsHash
	response.TxidsId = TxidsId
	response.TxidsAddress = TxidsAddress
	response.CallResults = callResults
	return nil
}
