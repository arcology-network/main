package types

import (
	"errors"
	"fmt"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
)

// type StdTransactionPack struct {
// 	Txs        []*types.StandardTransaction
// 	Src        types.TxSource
// 	TxHashChan chan evmCommon.Hash
// }

func ToStdTransaction(tx []byte, txfrom byte) (*types.StandardTransaction, error) {
	txType := tx[0]
	txReal := tx[1:]
	switch txType {
	case types.TxType_Eth:
		otx := new(evmTypes.Transaction)
		if err := otx.UnmarshalBinary(txReal); err != nil {
			return nil, err
		}
		txhash := types.RlpHash(otx)

		checkingTx := types.StandardTransaction{
			TxHash:            txhash,
			NativeTransaction: otx,
			TxRawData:         tx,
			Source:            txfrom,
		}

		return &checkingTx, nil
	}

	return &types.StandardTransaction{}, errors.New("tx type not defined")
}

func ToStdTransactionWorker(start, end, idx int, args ...interface{}) {
	txs := args[0].([]interface{})[0].([][]byte)
	transactions := args[0].([]interface{})[1].(*[]*types.StandardTransaction)
	logg := args[0].([]interface{})[2].(*actor.WorkerThreadLogger)

	for i, tx := range txs[start:end] {
		transaction, err := ToStdTransaction(tx[1:], tx[0])
		if err != nil {
			logg.Log(log.LogLevel_Error, "received block tx ", zap.Int("idx", i+start), zap.String("err", err.Error()), zap.String("tx", fmt.Sprintf("%x", tx)), zap.String("from", fmt.Sprintf("%x", tx[0])))
			continue
		}
		(*transactions)[i+start] = transaction
	}
}

func NewPack(txs [][]byte, src types.TxSource, hasChan bool, concurrency int, interLog *actor.WorkerThreadLogger) *types.StdTransactionPack {
	txLen := len(txs)
	stdTransactions := make([]*types.StandardTransaction, txLen)
	common.ParallelWorker(txLen, concurrency, ToStdTransactionWorker, txs, &stdTransactions, interLog)

	pack := types.StdTransactionPack{
		Txs: stdTransactions,
		Src: src,
	}

	if hasChan {
		pack.TxHashChan = make(chan evmCommon.Hash, 1)
	}
	return &pack
}
