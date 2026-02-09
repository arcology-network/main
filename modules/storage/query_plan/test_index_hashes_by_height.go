package queryplan

import (
	"math/big"

	"github.com/arcology-network/streamer/query"
)

type TestHashesByHeightQueryPlan struct {
}

func (p *TestHashesByHeightQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestHashesByHeightQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Height] = ctx.Req.(*big.Int)
			cont(nil, nil)
		},
	}
	getHashes := &query.CallStep{
		Plan: BuildGetBlockHashesByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Hashes] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			tx := ctx.Vars[QueryKey_Hashes]
			cont(tx, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getHashes,
			returnTx,
		},
	}
}
