package queryplan

import (
	"fmt"

	"github.com/arcology-network/main/modules/storage/query"
	mtypes "github.com/arcology-network/main/types"
)

type CodeQueryPlan struct {
}

func (p *CodeQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *CodeQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestParameters)
			ctx.Vars[QueryKey_AccountAddress] = fmt.Sprintf("%x", request.Address.Bytes())
			cont(nil, nil)
		},
	}
	getCode := &query.CallStep{
		Plan: BuildGetCodeSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Code] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			code := ctx.Vars[QueryKey_Code]
			cont(code, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getCode,
			returnTx,
		},
	}
}
