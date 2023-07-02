package types

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	ccurl "github.com/arcology-network/concurrenturl"
	"github.com/arcology-network/concurrenturl/interfaces"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	evmCommon "github.com/arcology-network/evm/common"
	evmTypes "github.com/arcology-network/evm/core/types"
	adaptor "github.com/arcology-network/vm-adaptor"
	ccapi "github.com/arcology-network/vm-adaptor/api"

	"go.uber.org/zap"
)

type ExecMessagers struct {
	Snapshot *interfaces.Datastore
	Config   *adaptor.Config
	Sequence *types.ExecutingSequence
	// Uuid     uint64
	// SerialID int
	// Total    int
	Debug bool
}

type ExecutionResponse struct {
	AccessRecords   []*types.TxAccessRecords
	EuResults       []*types.EuResult
	Receipts        []*evmTypes.Receipt
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
	ExecutingLogs   []string
}

type ExecutionService struct {
	Eu  *adaptor.EU
	Api *ccapi.API
	DB  *interfaces.Datastore
	Url *ccurl.ConcurrentUrl
}

func (es *ExecutionService) Init(eu *adaptor.EU, url *ccurl.ConcurrentUrl) {
	es.Eu = eu
	es.Url = url
}

func (es *ExecutionService) SetDB(db *interfaces.Datastore) {
	es.DB = db
}

// coinbase evmCommon.Address
func (es *ExecutionService) Exec(sequence *types.ExecutingSequence, config *adaptor.Config, logg *actor.WorkerThreadLogger, gatherExeclog bool) (*ExecutionResponse, error) {
	//gatherExeclog := viper.GetBool("execlog")

	response := ExecutionResponse{}
	msgLen := len(sequence.Msgs)
	response.EuResults = make([]*types.EuResult, 0, msgLen*2)
	response.AccessRecords = make([]*types.TxAccessRecords, 0, msgLen*2)
	response.Receipts = make([]*evmTypes.Receipt, 0, msgLen*2)
	logg.Log(log.LogLevel_Info, "service.Exec", zap.Int("nums", msgLen))
	failed := 0
	// es.TxsQueue.Reset(sequence.Msgs, sequence.Txids)
	// spawnedHashes := make(map[evmCommon.Hash]evmCommon.Hash, msgLen)
	// txids := make(map[evmCommon.Hash]*DefItem, msgLen)
	// hashList := make([]evmCommon.Hash, 0, msgLen*2)
	contractAddress := []evmCommon.Address{}
	nilAddress := evmCommon.Address{}
	response.CallResults = make([][]byte, 0, msgLen*2)
	response.ExecutingLogs = make([]string, 0, msgLen*2)

	for i, m := range sequence.Msgs {
		id := sequence.Txids[i]

		es.Url = ccurl.NewConcurrentUrl(*es.DB)
		es.Api = ccapi.NewAPI(es.Url)
		// statedb := eth.NewImplStateDB(es.Api)
		// es.Eu.SetContext(statedb, es.Api)

		th := evmCommon.BytesToHash(m.TxHash[:])
		// hashList = append(hashList, m.TxHash)
		msg := m.Native
		receipt, callResult, err := es.Eu.Run(th, uint32(id), msg, adaptor.NewEVMBlockContext(config), adaptor.NewEVMTxContext(*msg))
		if err != nil {

		}
		accesses, transitions := es.Url.ExportAll()
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
		result.ID = id

		if receipt.Status == 0 {
			failed = failed + 1
		}

		result.Transitions = univaluepk.UnivaluesEncode(transitions)
		// result.Transitions = append(result.Transitions, nonceTransitions...)

		// if result.DC != nil {
		// 	result.DC.ContractAddress = types.Address(evmCommon.HexToAddress(string(result.DC.ContractAddress)).Bytes())
		// }
		accessRecord := types.TxAccessRecords{}
		accessRecord.Hash = result.H
		accessRecord.ID = id

		accessRecord.Accesses = univaluepk.UnivaluesEncode(accesses)

		if !sequence.Parallel {
			// if result.DC != nil {
			// 	hs := [][]byte{m.TxHash[:], common.Uint64ToBytes(m.Native.Nonce)}
			// 	hash := sha256.Sum256(common.Flatten(hs))
			// 	standardMessager := types.MakeMessageWithDefCall(result.DC, hash, m.Native.Nonce)
			// 	txid := id + 1
			// 	es.TxsQueue.Insert(standardMessager, txid)
			// 	spawnedHashes[m.TxHash] = standardMessager.TxHash

			// 	// txids[m.TxHash] = &DefItem{
			// 	txids[standardMessager.TxHash] = &DefItem{
			// 		ContractAddress: evmCommon.BytesToAddress([]byte(result.DC.ContractAddress)),
			// 		Id:              txid,
			// 	}
			// }
			es.Url.Init(*es.DB)
			//transitionData := univaluepk.UnivaluesDecode(transitions, func() interface{} { return &univaluepk.Univalue{} }, nil)
			es.Url.Import(transitions)
			es.Url.Sort()
			es.Url.Finalize([]uint32{id})
			es.Url.WriteToDbBuffer()
			es.Url.SaveToDB()
		}

		response.EuResults = append(response.EuResults, &result)
		response.AccessRecords = append(response.AccessRecords, &accessRecord)
		// ethReceipt := ConvertReceipt(receipt)
		response.Receipts = append(response.Receipts, receipt)
		if receipt.ContractAddress != nilAddress {
			contractAddress = append(contractAddress, receipt.ContractAddress)
			//logg.Log(log.LogLevel_Debug, "debug contractAddress", zap.String("contractAddress", fmt.Sprintf("%x", ethReceipt.ContractAddress)))
		}
		//logg.Log(log.LogLevel_Debug, "debug exec result", zap.String("hash", fmt.Sprintf("%x", []byte(result.H))), zap.Int("transitions", len(transitions)), zap.Uint64("nonce", m.Native.Nonce()), zap.Uint64("Success", ethReceipt.Status))
		response.CallResults = append(response.CallResults, callResult.ReturnData)
	}
	response.ContractAddress = contractAddress
	// response.SpawnHashes = spawnedHashes
	// relation := map[evmCommon.Hash][]evmCommon.Hash{}
	// if !sequence.Parallel {
	// 	relation[sequence.SequenceId] = hashList
	// }
	// response.Relation = relation
	// response.Txids = txids
	logg.Log(log.LogLevel_Info, "*********************************service.Exec end", zap.Int("nums", msgLen), zap.Int("failed", failed))
	return &response, nil
}
