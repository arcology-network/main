package queryplan

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type TransactionCountQueryPlan struct {
	GetLastHeight func() uint64
}

func (p *TransactionCountQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}

func (p *TransactionCountQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestParameters)
			ctx.Vars[QueryKey_Address] = request.Address
			ctx.Vars[QueryKey_BlockParams] = request.BlockParams
			cont(nil, nil)
		},
	}
	getHeight := &query.CallStep{
		Plan: BuildGetHeightByHashOrNumberSubPlan(p.GetLastHeight),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Height] = v
		},
	}

	getState := &query.CallStep{
		Plan: BuildGetStateRootByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_StateRoot] = v
		},
	}
	getNonce := &query.CallStep{
		Plan: BuildGetNonceSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Nonce] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			nonce := ctx.Vars[QueryKey_Nonce]
			cont(nonce, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getHeight,
			getState,
			getNonce,
			returnTx,
		},
	}
}
