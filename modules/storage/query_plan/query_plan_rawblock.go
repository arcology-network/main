package queryplan

import (
	"fmt"
	"math/big"

	"github.com/arcology-network/streamer/query"
)

type GetRawBlockQueryPlan struct {
}

func (p *GetRawBlockQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *GetRawBlockQueryPlan) buildSteps() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Height] = big.NewInt(0).SetUint64(ctx.Req.(uint64))

			cont(nil, nil)
		},
	}
	getBlock := &query.CallStep{
		Plan: BuildGetBlockByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_MonacoBlock] = v
		},
	}

	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			v := ctx.Vars[QueryKey_MonacoBlock]
			if v == nil {
				cont(nil, fmt.Errorf("block not found"))
				return
			}
			cont(v, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			callGetBlockParam,
			getBlock,
			finalReturn,
		},
	}
}
