package types

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/core"
	evmTypes "github.com/arcology-network/evm/core/types"
	evmRlp "github.com/arcology-network/evm/rlp"
	"go.uber.org/zap"
)

type CheckingTxsPack struct {
	Txs        []*CheckingTx
	Src        types.TxSource
	TxHashChan chan evmCommon.Hash
}

type CheckingTx struct {
	Message     types.StandardMessage
	Transaction types.StandardTransaction
}

func (ctx *CheckingTx) UnSign(chainID *big.Int) error {
	otx := ctx.Transaction.Native
	msg, err := core.TransactionToMessage(otx, evmTypes.NewEIP155Signer(chainID), nil)
	if err != nil {
		return err
	}
	// msg.SkipAccountChecks = true
	ctx.Message.Native = msg
	return nil
}

func NewCheckingTxHash(tx []byte, txfrom byte) (*CheckingTx, error) {
	txType := tx[0]
	txReal := tx[1:]
	switch txType {
	case types.TxType_Eth:
		otx := new(evmTypes.Transaction)
		if err := evmRlp.DecodeBytes(txReal, otx); err != nil {
			return nil, err
		}
		txhash := types.RlpHash(otx)

		checkingTx := CheckingTx{
			Message: types.StandardMessage{
				TxHash:    txhash,
				Source:    txfrom,
				TxRawData: tx,
			},
			Transaction: types.StandardTransaction{
				TxHash:    txhash,
				Native:    otx,
				TxRawData: tx,
				Source:    txfrom,
			},
		}
		return &checkingTx, nil
	}

	return &CheckingTx{}, errors.New("tx type not defined")
}

func CheckingTxHashWorker(start, end, idx int, args ...interface{}) {
	txs := args[0].([]interface{})[0].([][]byte)
	checks := args[0].([]interface{})[1].(*[]*CheckingTx)
	logg := args[0].([]interface{})[2].(*actor.WorkerThreadLogger)

	for i, tx := range txs[start:end] {
		checkingTx, err := NewCheckingTxHash(tx[1:], tx[0])
		if err != nil {
			logg.Log(log.LogLevel_Error, "received block tx ", zap.Int("idx", i+start), zap.String("err", err.Error()), zap.String("tx", fmt.Sprintf("%x", tx)), zap.String("from", fmt.Sprintf("%x", tx[0])))
			continue
		}
		(*checks)[i+start] = checkingTx
	}
}
