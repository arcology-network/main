package queryplan

import (
	"github.com/arcology-network/main/modules/storage/query"
	mtypes "github.com/arcology-network/main/types"
)

type TestGetHeightByHashOrNumberQueryPlan struct {
	GetLastHeight func() uint64
}

func (p *TestGetHeightByHashOrNumberQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TestGetHeightByHashOrNumberQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			blockParams := ctx.Req.(*mtypes.BlockNumberOrHash)
			ctx.Vars[QueryKey_BlockParams] = blockParams
			cont(nil, nil)
		},
	}
	getHeight := &query.CallStep{
		Plan: BuildGetHeightByHashOrNumberSubPlan(p.GetLastHeight),
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
