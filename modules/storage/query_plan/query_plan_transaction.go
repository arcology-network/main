package queryplan

import (
	"math/big"

	"github.com/arcology-network/main/modules/storage/query"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type TransactionQueryPlan struct {
	GetChainId func() *big.Int
}

func (p *TransactionQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.build()
	query.StartStep(ctx, root, cont)
}

func (p *TransactionQueryPlan) build() query.Step {
	parseArgs := &query.FuncStep{
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

	// ensurePosition := &query.ReturnFromSubPlanStep{
	// 	Cond: func(ctx *query.QueryContext) bool {
	// 		return ctx.Vars[QueryKey_Position] == nil
	// 	},
	// 	Value: func(ctx *query.QueryContext) any {
	// 		return nil
	// 	},
	// 	Err: query.ErrHashNotFound,
	// }

	// ensurePosition := FuncEnsure(
	// 	func(ctx *query.QueryContext) bool {
	// 		return ctx.Vars[QueryKey_Position] == nil
	// 	},
	// 	query.ErrHashNotFound,
	// )

	ensurePosition := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			pos := ctx.Vars[QueryKey_Position]
			if pos == nil || pos.(*mstypes.Position) == nil {
				cont(nil, query.ErrHashNotFound)
				return
			}
			cont(nil, nil)
		},
	}

	callGetBlockParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			pos := ctx.Vars[QueryKey_Position].(*mstypes.Position)
			ctx.Vars[QueryKey_Height] = new(big.Int).SetUint64(pos.Height)
			ctx.Vars[QueryKey_Fulltx] = false
			ctx.Vars[QueryKey_OnlyHeader] = true
			ctx.Vars[QueryKey_ChainID] = p.GetChainId()
			cont(nil, nil)
		},
	}

	getBlock := &query.CallStep{
		Plan: BuildGetRpcBlockSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcBlockResult] = v
		},
	}

	callGetTransactionParam := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			blockResult := ctx.Vars[QueryKey_RpcBlockResult].(*RpcBlockResult)
			ctx.Vars[QueryKey_BlockHash] = blockResult.Block.Header.Hash()
			ctx.Vars[QueryKey_SignerType] = blockResult.Signer
			cont(nil, nil)
		},
	}
	// ctx.Vars["blockHash"]
	getTx := &query.CallStep{
		Plan: BuildGetTransactionByPositionSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_RpcTransaction] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			cont(ctx.Vars[QueryKey_RpcTransaction], nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			parseArgs,
			getPosition,
			ensurePosition,
			callGetBlockParam,
			getBlock,
			callGetTransactionParam,
			getTx,
			returnTx,
		},
	}
}
