package ethapi

import (
	"math/big"
	"time"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	evmCommon "github.com/arcology-network/evm/common"
	ethTypes "github.com/arcology-network/evm/core/types"
	internal "github.com/arcology-network/main/modules/eth-api/backend"
)

type FilterManager struct {
	actor.WorkerThread
	filters *internal.Filters
}

// return a Subscriber struct
func NewFilterManager(concurrency int, groupid string) actor.IWorkerEx {
	fm := FilterManager{}
	fm.Set(concurrency, groupid)
	return &fm
}

func (fm *FilterManager) Inputs() ([]string, bool) {
	return []string{
		actor.MsgSelectedReceipts,
		actor.MsgPendingBlock,
	}, true
}

func (fm *FilterManager) Outputs() map[string]int {
	return map[string]int{}
}

func (fm *FilterManager) Config(params map[string]interface{}) {
	fm.filters = internal.NewFilters()
	fm.filters.SetTimeout(time.Minute * time.Duration(int(params["filter_timeout_mins"].(float64))))

	options.KeyFile = params["key_file"].(string)
	options.Port = uint64(params["json_rpc_port"].(float64))
	options.Debug = params["debug"].(bool)
	options.Waits = int(params["retry_time"].(float64))
	options.Coinbase = params["coinbase"].(string)
	options.ProtocolVersion = int(params["protocol_version"].(float64))
	options.Hashrate = int(params["hash_rate"].(float64))
}

func (*FilterManager) OnStart() {
	startJsonRpc()
}

func (*FilterManager) Stop() {}

func (fm *FilterManager) OnMessageArrived(msgs []*actor.Message) error {
	var receipts []*ethTypes.Receipt
	var block *types.MonacoBlock

	for _, v := range msgs {
		switch v.Name {
		case actor.MsgSelectedReceipts:
			for _, item := range v.Data.([]interface{}) {
				receipts = append(receipts, item.(*ethTypes.Receipt))
			}
		case actor.MsgPendingBlock:
			block = v.Data.(*types.MonacoBlock)
		}
	}

	blockHash := block.Hash()
	worker := func(start, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			receipts[i].BlockHash = evmCommon.BytesToHash(blockHash)
			receipts[i].BlockNumber = big.NewInt(int64(block.Height))
			receipts[i].TransactionIndex = uint(i)

			for k := range receipts[i].Logs {
				receipts[i].Logs[k].BlockHash = receipts[i].BlockHash
				receipts[i].Logs[k].TxHash = receipts[i].TxHash
				receipts[i].Logs[k].TxIndex = receipts[i].TransactionIndex
			}
			//storageTypes.SaveReceipt(s.datastore, block.Height, txhash, (*receipts)[i])
		}
	}

	common.ParallelWorker(len(receipts), fm.Concurrency, worker)
	fm.filters.OnResultsArrived(block.Height, receipts, evmCommon.BytesToHash(blockHash))

	return nil
}
