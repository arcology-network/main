package queryplan

import (
	"math/big"

	"github.com/arcology-network/streamer/query"
)

type TestReceiptsByHeightQueryPlan struct {
}

func (p *TestReceiptsByHeightQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}

func (p *TestReceiptsByHeightQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Height] = ctx.Req.(*big.Int)
			cont(nil, nil)
		},
	}
	getReceipts := &query.CallStep{
		Plan: BuildGetReceiptsByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Receipts] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			rs := ctx.Vars[QueryKey_Receipts]
			cont(rs, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getReceipts,
			returnTx,
		},
	}
}
