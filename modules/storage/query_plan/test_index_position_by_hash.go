package queryplan

import (
	"github.com/arcology-network/main/modules/storage/query"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TestPositionByHashQueryPlan struct {
}

func (p *TestPositionByHashQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestPositionByHashQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_TxHash] = ctx.Req.(evmCommon.Hash)
			cont(nil, nil)
		},
	}
	getPosition := &query.CallStep{
		Plan: BuildGetPositionByHashSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Position] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			p := ctx.Vars[QueryKey_Position]
			cont(p, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getPosition,
			returnTx,
		},
	}
}
