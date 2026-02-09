package queryplan

import (
	"math/big"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type TxByNumberAndIdxQueryPlan struct {
	GetChainId     func() *big.Int
	GetQueryHeight func(height int64) int64
}

func (p *TxByNumberAndIdxQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *TxByNumberAndIdxQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestBlockEth)

			queryHeight := p.GetQueryHeight(request.Number)

			pos := mstypes.Position{
				Height:     uint64(queryHeight),
				IdxInBlock: request.Index,
			}
			ctx.Vars[QueryKey_Height] = big.NewInt(queryHeight)
			ctx.Vars[QueryKey_Position] = &pos

			ctx.Vars[QueryKey_Fulltx] = false
			ctx.Vars[QueryKey_OnlyHeader] = true
			ctx.Vars[QueryKey_BlockHash] = request.Hash
			ctx.Vars[QueryKey_IdxInBlock] = request.Index
			ctx.Vars[QueryKey_ChainID] = p.GetChainId()
			cont(nil, nil)
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
			getBlock,
			callGetTransactionParam,
			getTx,
			returnTx,
		},
	}
}
