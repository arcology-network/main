package queryplan

import (
	"github.com/arcology-network/main/modules/storage/query"
)

type TestGetTransactionByPositionQueryPlan struct {
}

func (p *TestGetTransactionByPositionQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestGetTransactionByPositionQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			params := ctx.Req.(*QueryParamTransactionByPosition)
			ctx.Vars[QueryKey_ChainID] = params.ChainId
			ctx.Vars[QueryKey_Position] = params.Position
			ctx.Vars[QueryKey_BlockHash] = params.BlockHash
			ctx.Vars[QueryKey_SignerType] = params.SignerType
			cont(nil, nil)
		},
	}
	gettx := &query.CallStep{
		Plan: BuildGetTransactionByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcTransaction] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			rp := ctx.Vars[QueryKey_RpcTransaction]
			cont(rp, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			gettx,
			returnTx,
		},
	}
}
