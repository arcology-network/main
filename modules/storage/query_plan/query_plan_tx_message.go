package queryplan

import (
	"math/big"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/modules/storage/query"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
)

type TxMessageQueryPlan struct {
	GetChainId func() *big.Int
}

func (p *TxMessageQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *TxMessageQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			hash := ctx.Req.(evmCommon.Hash)
			ctx.Vars[QueryKey_Fulltx] = false
			ctx.Vars[QueryKey_OnlyHeader] = true
			ctx.Vars[QueryKey_TxHash] = hash
			ctx.Vars[QueryKey_ChainID] = p.GetChainId()
			cont(nil, nil)
		},
	}

	getHeight := &query.CallStep{
		Plan: BuildGetPositionByHashSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			pos := v.(*mstypes.Position)
			ctx.Vars[QueryKey_Height] = big.NewInt(0).SetUint64(pos.Height)
			ctx.Vars[QueryKey_Position] = pos
		},
	}

	getTx := &query.CallStep{
		Plan: BuildGetTxByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Transaction] = v.(*evmTypes.Transaction)
		},
	}

	getBlock := &query.CallStep{
		Plan: BuildGetRpcBlockSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcBlockResult] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			txHash := ctx.Vars[QueryKey_TxHash].(evmCommon.Hash)
			pos := ctx.Vars[QueryKey_Position].(*mstypes.Position)
			blockResult := ctx.Vars[QueryKey_RpcBlockResult].(*RpcBlockResult)
			ChainId := ctx.Vars[QueryKey_ChainID].(*big.Int)

			standardTransaction := types.StandardTransaction{
				TxHash:            txHash,
				NativeTransaction: ctx.Vars[QueryKey_Transaction].(*evmTypes.Transaction),
				Signer:            blockResult.Signer,
			}
			signer := mtypes.MakeSigner(blockResult.Signer, ChainId)
			err := standardTransaction.UnSign(signer)
			if err != nil {
				cont(nil, err)
				return
			}

			cont(&mtypes.QueryReplayMsgResult{
				Msg: standardTransaction.NativeMessage,
				Ctx: &tracers.Context{
					BlockHash:   blockResult.Block.Header.TxHash,
					BlockNumber: big.NewInt(int64(pos.Height)),
					TxIndex:     pos.IdxInBlock,
					TxHash:      txHash,
				},
			}, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			callGetBlockParam,
			getHeight,
			getTx,
			getBlock,
			returnTx,
		},
	}
}
