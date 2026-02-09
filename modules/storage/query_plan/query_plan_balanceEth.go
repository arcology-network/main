package queryplan

import (
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type BalanceEthQueryPlan struct {
}

func (p *BalanceEthQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *BalanceEthQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestParameters)
			ctx.Vars[QueryKey_AccountAddress] = fmt.Sprintf("%x", request.Address.Bytes())
			cont(nil, nil)
		},
	}
	getBal := &query.CallStep{
		Plan: BuildGetBalanceSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Balance] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			balance := ctx.Vars[QueryKey_Balance]
			cont(balance, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getBal,
			returnTx,
		},
	}
}
