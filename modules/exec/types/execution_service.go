package types

import (
	"crypto/sha256"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	ethTypes "github.com/arcology-network/3rd-party/eth/types"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	ccurl "github.com/arcology-network/concurrenturl/v2"
	urlcommon "github.com/arcology-network/concurrenturl/v2/common"
	ccurlTypes "github.com/arcology-network/concurrenturl/v2/type"
	mevmCommon "github.com/arcology-network/evm/common"
	mevmTypes "github.com/arcology-network/evm/core/types"
	adaptor "github.com/arcology-network/vm-adaptor/evm"
	"go.uber.org/zap"
)

type ExecMessagers struct {
	Snapshot *urlcommon.DatastoreInterface
	Config   *adaptor.Config
	Sequence *types.ExecutingSequence
	Uuid     uint64
	SerialID int
	Total    int
	Debug    bool
}

func ConvertMessage(msg *types.StandardMessage) *mevmTypes.Message {

	var message mevmTypes.Message
	if msg.Native.To() != nil {
		to := mevmCommon.BytesToAddress(msg.Native.To().Bytes())
		message = mevmTypes.NewMessage(
			mevmCommon.BytesToAddress(msg.Native.From().Bytes()),
			&to,
			msg.Native.Nonce(),
			msg.Native.Value(),
			msg.Native.Gas(),
			msg.Native.GasPrice(),
			msg.Native.Data(),
			mevmTypes.AccessList{},
			//msg.Native.CheckNonce(),
			false,
		)
	} else {
		message = mevmTypes.NewMessage(
			mevmCommon.BytesToAddress(msg.Native.From().Bytes()),
			nil,
			msg.Native.Nonce(),
			msg.Native.Value(),
			msg.Native.Gas(),
			msg.Native.GasPrice(),
			msg.Native.Data(),
			mevmTypes.AccessList{},
			//msg.Native.CheckNonce(),
			false,
		)
	}
	return &message
}

type ExecutionResponse struct {
	AccessRecords   []*types.TxAccessRecords
	EuResults       []*types.EuResult
	Receipts        []*ethTypes.Receipt
	Relation        map[ethCommon.Hash][]ethCommon.Hash
	SpawnHashes     map[ethCommon.Hash]ethCommon.Hash
	ContractAddress []ethCommon.Address
	Txids           map[ethCommon.Hash]*DefItem
	CallResults     [][]byte
	ExecutingLogs   []string
}

type DefItem struct {
	ContractAddress ethCommon.Address
	Id              uint32
}

type ExecutionService struct {
	Eu       *adaptor.EUV2
	TxsQueue *Queue
	Api      *adaptor.APIV2
	DB       *urlcommon.DatastoreInterface
	Url      *ccurl.ConcurrentUrl
}

func ConvertReceipt(ereceipt *mevmTypes.Receipt) *ethTypes.Receipt {
	dreceipt := ethTypes.Receipt{}
	if ereceipt != nil {
		dreceipt.PostState = ereceipt.PostState
		dreceipt.Status = ereceipt.Status
		dreceipt.CumulativeGasUsed = ereceipt.CumulativeGasUsed
		dreceipt.Bloom = ethTypes.BytesToBloom(ereceipt.Bloom.Bytes())
		logs := make([]*ethTypes.Log, len(ereceipt.Logs))
		for i, log := range ereceipt.Logs {
			topics := make([]ethCommon.Hash, len(log.Topics))
			for j, topic := range log.Topics {
				topics[j] = ethCommon.BytesToHash(topic.Bytes())
			}
			logs[i] = &ethTypes.Log{
				Address:     ethCommon.BytesToAddress(log.Address.Bytes()),
				Topics:      topics,
				Data:        log.Data,
				BlockNumber: log.BlockNumber,
				TxHash:      ethCommon.BytesToHash(log.TxHash.Bytes()),
				TxIndex:     log.TxIndex,
				BlockHash:   ethCommon.BytesToHash(log.BlockHash.Bytes()),
				Index:       log.Index,
				Removed:     log.Removed,
			}
		}
		dreceipt.Logs = logs
		dreceipt.TxHash = ethCommon.BytesToHash(ereceipt.TxHash.Bytes())
		dreceipt.ContractAddress = ethCommon.BytesToAddress(ereceipt.ContractAddress.Bytes())
		dreceipt.GasUsed = ereceipt.GasUsed

		dreceipt.Type = ereceipt.Type
	}
	return &dreceipt
}

func (es *ExecutionService) Init(eu *adaptor.EUV2, url *ccurl.ConcurrentUrl) {
	es.Eu = eu
	es.TxsQueue = NewQueue()
	es.Url = url
}

func (es *ExecutionService) SetDB(db *urlcommon.DatastoreInterface) {
	es.DB = db
}

// coinbase mevmCommon.Address
func (es *ExecutionService) Exec(sequence *types.ExecutingSequence, config *adaptor.Config, logg *actor.WorkerThreadLogger, gatherExeclog bool) (*ExecutionResponse, error) {
	//gatherExeclog := viper.GetBool("execlog")

	response := ExecutionResponse{}
	msgLen := len(sequence.Msgs)
	response.EuResults = make([]*types.EuResult, 0, msgLen*2)
	response.AccessRecords = make([]*types.TxAccessRecords, 0, msgLen*2)
	response.Receipts = make([]*ethTypes.Receipt, 0, msgLen*2)
	logg.Log(log.LogLevel_Info, "service.Exec", zap.Int("nums", msgLen))
	failed := 0
	es.TxsQueue.Reset(sequence.Msgs, sequence.Txids)
	spawnedHashes := make(map[ethCommon.Hash]ethCommon.Hash, msgLen)
	txids := make(map[ethCommon.Hash]*DefItem, msgLen)
	hashList := make([]ethCommon.Hash, 0, msgLen*2)
	contractAddress := []ethCommon.Address{}
	nilAddress := ethCommon.Address{}
	response.CallResults = make([][]byte, 0, msgLen*2)
	response.ExecutingLogs = make([]string, 0, msgLen*2)
	for {
		m, id := es.TxsQueue.GetNext()
		if m == nil {
			break
		}
		//es.Url.Init(*es.DB)
		es.Url = ccurl.NewConcurrentUrl(*es.DB)

		es.Api = adaptor.NewAPIV2(*es.DB, es.Url)
		statedb := adaptor.NewStateDBV2(es.Api, *es.DB, es.Url)
		es.Eu.SetContext(statedb, es.Api, *es.DB, es.Url)

		th := mevmCommon.BytesToHash(m.TxHash[:])
		hashList = append(hashList, m.TxHash)
		msg := ConvertMessage(m)
		accesses, transitions, nonceTransitions, receipt, callResult := es.Eu.RunEx(th, int(id), msg, adaptor.NewEVMBlockContextV2(config), adaptor.NewEVMTxContext(*msg))
		if gatherExeclog {
			logs := es.Api.GetLogs()
			clogs := make([]types.ExecutingLog, len(logs))
			for j, ilog := range logs {
				clogs[j] = types.ExecutingLog{
					Key:   ilog.GetKey(),
					Value: ilog.GetValue(),
				}
			}

			executingLogs := types.ExecutingLogs{
				Logs:   clogs,
				Txhash: m.TxHash,
			}
			jsonExecutingLogs, err := executingLogs.Marshal()
			if err == nil {
				response.ExecutingLogs = append(response.ExecutingLogs, jsonExecutingLogs)
			}
		}
		es.Api.ClearLogs()
		result := types.EuResult{}
		result.H = string(th.Bytes())
		result.GasUsed = receipt.GasUsed
		result.Status = receipt.Status
		result.DC = es.Api.GetDeferCall()
		result.ID = id

		if receipt.Status == 0 {
			failed = failed + 1
		}

		result.Transitions = transitions
		result.Transitions = append(result.Transitions, nonceTransitions...)

		if result.DC != nil {
			result.DC.ContractAddress = types.Address(ethCommon.HexToAddress(string(result.DC.ContractAddress)).Bytes())
		}
		accessRecord := types.TxAccessRecords{}
		accessRecord.Hash = result.H
		accessRecord.ID = id

		accessRecord.Accesses = accesses

		if !sequence.Parallel {
			if result.DC != nil {
				hs := [][]byte{m.TxHash[:], common.Uint64ToBytes(m.Native.Nonce())}
				hash := sha256.Sum256(common.Flatten(hs))
				standardMessager := types.MakeMessageWithDefCall(result.DC, hash, m.Native.Nonce())
				txid := id + 1
				es.TxsQueue.Insert(standardMessager, txid)
				spawnedHashes[m.TxHash] = standardMessager.TxHash

				txids[m.TxHash] = &DefItem{
					ContractAddress: ethCommon.BytesToAddress([]byte(result.DC.ContractAddress)),
					Id:              txid,
				}
			}
			es.Url.Init(*es.DB)
			transitionData := ccurlTypes.Univalues{}.DecodeV2(transitions, func() interface{} { return &ccurlTypes.Univalue{} }, nil)
			es.Url.Import(transitionData)
			es.Url.PostImport()
			es.Url.Precommit([]uint32{id})
			es.Url.Postcommit()
			es.Url.SaveToDB()
		}

		response.EuResults = append(response.EuResults, &result)
		response.AccessRecords = append(response.AccessRecords, &accessRecord)
		ethReceipt := ConvertReceipt(receipt)
		response.Receipts = append(response.Receipts, ethReceipt)
		if ethReceipt.ContractAddress != nilAddress {
			contractAddress = append(contractAddress, ethReceipt.ContractAddress)
			//logg.Log(log.LogLevel_Debug, "debug contractAddress", zap.String("contractAddress", fmt.Sprintf("%x", ethReceipt.ContractAddress)))
		}
		//logg.Log(log.LogLevel_Debug, "debug exec result", zap.String("hash", fmt.Sprintf("%x", []byte(result.H))), zap.Int("transitions", len(transitions)), zap.Uint64("nonce", m.Native.Nonce()), zap.Uint64("Success", ethReceipt.Status))
		response.CallResults = append(response.CallResults, callResult)
	}
	response.ContractAddress = contractAddress
	response.SpawnHashes = spawnedHashes
	relation := map[ethCommon.Hash][]ethCommon.Hash{}
	if !sequence.Parallel {
		relation[sequence.SequenceId] = hashList
	}
	response.Relation = relation
	response.Txids = txids
	logg.Log(log.LogLevel_Info, "*********************************service.Exec end", zap.Int("nums", msgLen), zap.Int("failed", failed))
	return &response, nil
}
