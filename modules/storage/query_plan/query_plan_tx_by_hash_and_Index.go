package queryplan

import (
	"math/big"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type TxByHashAndIdxQueryPlan struct {
	GetChainId func() *big.Int
}

func (p *TxByHashAndIdxQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *TxByHashAndIdxQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestBlockEth)
			ctx.Vars[QueryKey_Fulltx] = false
			ctx.Vars[QueryKey_OnlyHeader] = true
			ctx.Vars[QueryKey_BlockHash] = request.Hash
			ctx.Vars[QueryKey_IdxInBlock] = request.Index
			ctx.Vars[QueryKey_ChainID] = p.GetChainId()
			cont(nil, nil)
		},
	}

	getHeight := &query.CallStep{
		Plan: BuildGetBlockHeightByHashSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			height := v.(*big.Int)
			ctx.Vars[QueryKey_Height] = v
			ctx.Vars[QueryKey_Position] = &mstypes.Position{
				Height:     height.Uint64(),
				IdxInBlock: ctx.Vars[QueryKey_IdxInBlock].(int),
			}
		},
	}

	getBlock := &query.CallStep{
		Plan: BuildGetRpcBlockSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcBlockResult] = v
		},
	}

	callGetTransactionParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			blockResult := ctx.Vars[QueryKey_RpcBlockResult].(*RpcBlockResult)
			ctx.Vars[QueryKey_BlockHash] = blockResult.Block.Header.Hash()
			ctx.Vars[QueryKey_SignerType] = blockResult.Signer
			cont(nil, nil)
		},
	}

	getTx := &query.CallStep{
		Plan: BuildGetTransactionByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcTransaction] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			cont(ctx.Vars[QueryKey_RpcTransaction], nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			callGetBlockParam,
			getHeight,
			getBlock,
			callGetTransactionParam,
			getTx,
			returnTx,
		},
	}
}
