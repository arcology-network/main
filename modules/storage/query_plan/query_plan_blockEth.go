package queryplan

import (
	"math/big"

	"github.com/arcology-network/main/modules/storage/query"
	mtypes "github.com/arcology-network/main/types"
)

type BlockEthQueryPlan struct {
	GetQueryHeight func(height int64) int64
	GetChainId     func() *big.Int
}

func (p *BlockEthQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *BlockEthQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestBlockEth)
			ctx.Vars[QueryKey_Height] = big.NewInt(p.GetQueryHeight(request.Number))
			ctx.Vars[QueryKey_Fulltx] = request.FullTx
			ctx.Vars[QueryKey_OnlyHeader] = false
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

	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			rbr := ctx.Vars[QueryKey_RpcBlockResult].(*RpcBlockResult)
			cont(rbr.Block, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			callGetBlockParam,
			getBlock,
			finalReturn,
		},
	}
}
