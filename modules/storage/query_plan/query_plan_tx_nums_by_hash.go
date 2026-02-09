package queryplan

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TxNumsByHashQueryPlan struct {
}

func (p *TxNumsByHashQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *TxNumsByHashQueryPlan) build() query.Step {
	callGetBlockParam := &query.FuncStep{
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

	getBlock := &query.CallStep{
		Plan: BuildGetBlockByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_MonacoBlock] = v
		},
	}

	txCounter := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			mblock := ctx.Vars[QueryKey_MonacoBlock].(*mtypes.MonacoBlock)
			ctx.Vars[QueryKey_TransactionCount] = len(mblock.Txs)
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
			getHeight,
			getBlock,
			txCounter,
			final,
		},
	}
}
