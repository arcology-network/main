package queryplan

import (
	mstypes "github.com/arcology-network/main/modules/storage/types"
	"github.com/arcology-network/streamer/query"
)

type TestTxByPositionQueryPlan struct {
}

func (p *TestTxByPositionQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}

func (p *TestTxByPositionQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Position] = ctx.Req.(*mstypes.Position)
			cont(nil, nil)
		},
	}
	getTx := &query.CallStep{
		Plan: BuildGetTxByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Transaction] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			tx := ctx.Vars[QueryKey_Transaction]
			cont(tx, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getTx,
			returnTx,
		},
	}
}
