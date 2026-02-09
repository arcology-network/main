package queryplan

import (
	mstypes "github.com/arcology-network/main/modules/storage/types"
	"github.com/arcology-network/streamer/query"
)

type TestReceiptByPositionQueryPlan struct {
}

func (p *TestReceiptByPositionQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestReceiptByPositionQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Position] = ctx.Req.(*mstypes.Position)
			cont(nil, nil)
		},
	}
	getReceipt := &query.CallStep{
		Plan: BuildGetReceiptByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Receipt] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			r := ctx.Vars[QueryKey_Receipt]
			cont(r, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getReceipt,
			returnTx,
		},
	}
}
