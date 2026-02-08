package queryplan

import (
	"math/big"

	"github.com/arcology-network/main/modules/storage/query"
	mtypes "github.com/arcology-network/main/types"
)

type HeaderByHashQueryPlan struct {
	GetChainId func() *big.Int
}

func (p *HeaderByHashQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *HeaderByHashQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestBlockEth)
			ctx.Vars[QueryKey_BlockHash] = request.Hash
			ctx.Vars[QueryKey_Fulltx] = false
			ctx.Vars[QueryKey_OnlyHeader] = true
			ctx.Vars[QueryKey_ChainID] = p.GetChainId()
			cont(nil, nil)
		},
	}

	getHeight := &query.CallStep{
		Plan: BuildGetBlockHeightByHashSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Height] = v
		},
	}

	getBlock := &query.CallStep{
		Plan: BuildGetRpcBlockSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcBlockResult] = v
		},
	}

	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			rbr := ctx.Vars[QueryKey_RpcBlockResult].(*RpcBlockResult)
			cont(rbr.Block, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			callGetBlockParam,
			getHeight,
			getBlock,
			finalReturn,
		},
	}
}
