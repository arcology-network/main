package queryplan

import (
	"github.com/arcology-network/streamer/query"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TestHeightByHashQueryPlan struct {
}

func (p *TestHeightByHashQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestHeightByHashQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_BlockHash] = ctx.Req.(evmCommon.Hash)
			cont(nil, nil)
		},
	}
	getHeight := &query.CallStep{
		Plan: BuildGetBlockHeightByHashSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Height] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			h := ctx.Vars[QueryKey_Height]
			cont(h, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getHeight,
			returnTx,
		},
	}
}
