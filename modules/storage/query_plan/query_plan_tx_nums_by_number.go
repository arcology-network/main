package queryplan

import (
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type TxNumsByNumberQueryPlan struct {
	GetQueryHeight func(height int64) int64
}

func (p *TxNumsByNumberQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *TxNumsByNumberQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_Height] = big.NewInt(p.GetQueryHeight(ctx.Req.(int64)))
			cont(nil, nil)
		},
	}

	getBlock := &query.CallStep{
		Plan: BuildGetBlockByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_MonacoBlock] = v
		},
	}

	txCounter := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			mblock := ctx.Vars[QueryKey_MonacoBlock].(*mtypes.MonacoBlock)
			if mblock == nil {
				ctx.Vars[QueryKey_TransactionCount] = 0
			} else {
				ctx.Vars[QueryKey_TransactionCount] = len(mblock.Txs)
			}
			cont(nil, nil)
		},
	}

	final := &query.ResultStep{
		Build: func(ctx *query.QueryContext) any {
			return ctx.Vars[QueryKey_TransactionCount]
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			callGetBlockParam,
			getBlock,
			txCounter,
			final,
		},
	}
}
