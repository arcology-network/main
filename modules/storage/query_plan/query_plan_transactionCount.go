package queryplan

import (
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type TransactionCountQueryPlan struct {
}

func (p *TransactionCountQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *TransactionCountQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestParameters)
			ctx.Vars[QueryKey_AccountAddress] = fmt.Sprintf("%x", request.Address.Bytes())
			cont(nil, nil)
		},
	}
	getNonce := &query.CallStep{
		Plan: BuildGetNonceSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Nonce] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			nonce := ctx.Vars[QueryKey_Nonce]
			cont(nonce, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getNonce,
			returnTx,
		},
	}
}
