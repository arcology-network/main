package queryplan

import (
	"github.com/arcology-network/streamer/query"
)

type TestRpcBlockQueryPlan struct {
}

func (p *TestRpcBlockQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestRpcBlockQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			param := ctx.Req.(*QueryParamRpcBlock)
			ctx.Vars[QueryKey_Height] = param.Height
			ctx.Vars[QueryKey_Fulltx] = param.Fulltx
			ctx.Vars[QueryKey_OnlyHeader] = param.OnlyHeader
			ctx.Vars[QueryKey_ChainID] = param.ChainId
			cont(nil, nil)
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
			rbr := ctx.Vars[QueryKey_RpcBlockResult]
			cont(rbr, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getBlock,
			returnTx,
		},
	}
}
