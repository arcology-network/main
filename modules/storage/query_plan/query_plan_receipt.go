package queryplan

import (
	"fmt"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	"github.com/arcology-network/streamer/query"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type ReceiptQueryPlan struct {
}

func (p *ReceiptQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}
func (p *ReceiptQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ctx.Vars[QueryKey_TxHash] = ctx.Req.(evmCommon.Hash)
			cont(nil, nil)
		},
	}

	getPos := &query.CallStep{
		Plan: BuildGetPositionByHashSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Position] = v
		},
	}

	positionIsnul := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			pos := ctx.Vars[QueryKey_Position]
			if pos == nil || pos.(*mstypes.Position) == nil {
				cont(nil, fmt.Errorf("receipt not found"))
				return
			}
			cont(nil, nil)
		},
	}

	getReceipt := &query.CallStep{
		Plan: BuildGetReceiptByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Receipt] = v
		},
	}

	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			r := ctx.Vars[QueryKey_Receipt]
			cont(r, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getPos,
			positionIsnul,
			getReceipt,
			finalReturn,
		},
	}
}
