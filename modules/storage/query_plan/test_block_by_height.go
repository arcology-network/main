package queryplan

import (
	"math/big"

	"github.com/arcology-network/main/modules/storage/query"
)

type TestBlockByHeightQueryPlan struct {
}

func (p *TestBlockByHeightQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestBlockByHeightQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Height] = ctx.Req.(*big.Int)
			cont(nil, nil)
		},
	}
	getBlock := &query.CallStep{
		Plan: BuildGetBlockByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_MonacoBlock] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			mblock := ctx.Vars[QueryKey_MonacoBlock]
			cont(mblock, nil)
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
