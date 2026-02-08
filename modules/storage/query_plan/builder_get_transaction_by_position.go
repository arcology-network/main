package queryplan

import (
	"math/big"

	"github.com/arcology-network/main/modules/storage/query"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type TxResult struct {
	Tx *mtypes.RPCTransaction
}

func BuildGetTransactionByPositionSubPlan() query.Step {
	getTx := &query.CallStep{
		Plan: BuildGetTxByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Transaction] = v
		},
	}

	buildTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			chainID := ctx.Vars[QueryKey_ChainID].(*big.Int)
			tx := ctx.Vars[QueryKey_Transaction].(*evmTypes.Transaction)
			pos := ctx.Vars[QueryKey_Position].(*mstypes.Position)
			blockHash := ctx.Vars[QueryKey_BlockHash].(evmCommon.Hash)

			signer := mtypes.MakeSigner(ctx.Vars[QueryKey_SignerType].(uint8), chainID)
			from, _ := evmTypes.Sender(signer, tx)
			v, r, s := tx.RawSignatureValues()

			result := &mtypes.RPCTransaction{
				Type:     hexutil.Uint64(tx.Type()),
				From:     from,
				Gas:      hexutil.Uint64(tx.Gas()),
				GasPrice: (*hexutil.Big)(tx.GasPrice()),
				Hash:     tx.Hash(),
				Input:    hexutil.Bytes(tx.Data()),
				Nonce:    hexutil.Uint64(tx.Nonce()),
				To:       tx.To(),
				Value:    (*hexutil.Big)(tx.Value()),
				V:        (*hexutil.Big)(v),
				R:        (*hexutil.Big)(r),
				S:        (*hexutil.Big)(s),
			}
			if blockHash != (evmCommon.Hash{}) {
				result.BlockHash = &blockHash
				result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(pos.Height))
				idx := uint64(pos.IdxInBlock)
				result.TransactionIndex = (*hexutil.Uint64)(&idx)
			}

			if v.Sign() == 0 && r.Sign() == 0 && s.Sign() == 0 { // pre-bedrock relayed tx does not have a signature
				result.ChainID = (*hexutil.Big)(new(big.Int).Set(chainID))
				// break
			}
			// if a legacy transaction has an EIP-155 chain id, include it explicitly
			if id := tx.ChainId(); id.Sign() != 0 {
				result.ChainID = (*hexutil.Big)(id)
			}

			ctx.Vars[QueryKey_RpcTransaction] = result
			cont(nil, nil)
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			rb := ctx.Vars[QueryKey_RpcTransaction]
			cont(rb, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getTx,
			buildTx,
			returnTx,
		},
	}
}
