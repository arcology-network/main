package queryplan

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type BlockReceiptsQueryPlan struct {
	GetLastHeight func() uint64
}

func (p *BlockReceiptsQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *BlockReceiptsQueryPlan) buildSteps() query.Step {
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

	getReceipts := &query.CallStep{
		Plan: BuildGetReceiptsByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Receipts] = v
		},
	}

	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			rs := ctx.Vars[QueryKey_Receipts]
			cont(rs, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getHeight,
			getReceipts,
			finalReturn,
		},
	}
}
