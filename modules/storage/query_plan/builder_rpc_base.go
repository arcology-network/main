package queryplan

import (
	"errors"
	"math/big"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/query"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

// urlstore
func BuildGetBalanceSubPlan() query.Step {
	getBalance := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			addr := ctx.Vars[QueryKey_AccountAddress].(string)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"urlstore", "GetBalance",
				addr,
				"QueryContinuationAction",
				actor.NMeta(
					"contId",
					root.RegisterCont(cont),
				),
			)

		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getBalance,
		},
	}
}
func BuildGetNonceSubPlan() query.Step {
	getNonce := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			addr := ctx.Vars[QueryKey_AccountAddress].(string)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"urlstore", "GetNonce",
				addr,
				"QueryContinuationAction",
				actor.NMeta(
					"contId",
					root.RegisterCont(cont),
				),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getNonce,
		},
	}
}

func BuildGetCodeSubPlan() query.Step {
	getCode := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			addr := ctx.Vars[QueryKey_AccountAddress].(string)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"urlstore", "GetCode",
				addr,
				"QueryContinuationAction",
				actor.NMeta(
					"contId",
					root.RegisterCont(cont),
				),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getCode,
		},
	}
}

func BuildGetStorageSubPlan() query.Step {
	getStorage := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			addr := ctx.Vars[QueryKey_AccountAddress].(string)
			key := ctx.Vars[QueryKey_StorageKey].(string)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"urlstore", "GetEthStorage",
				&mtypes.UrlEthStorageGetRequest{
					Address: addr,
					Key:     key,
				},
				"QueryContinuationAction",
				actor.NMeta(
					"contId",
					root.RegisterCont(cont),
				),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getStorage,
		},
	}
}

// blockstore
func BuildGetBlockByHeightSubPlan() query.Step {
	getBlock := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			height := ctx.Vars[QueryKey_Height].(*big.Int)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"blockstore", "GetByHeight",
				height.Uint64(),
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getBlock,
		},
	}
}

func BuildGetTxByPositionSubPlan() query.Step {
	getTx := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			pos := ctx.Vars[QueryKey_Position].(*mstypes.Position)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"blockstore", "GetTransaction",
				pos,
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getTx,
		},
	}
}

// indexstore
func BuildGetBlockHashesByHeightSubPlan() query.Step {
	getBlockHashes := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			height := ctx.Vars[QueryKey_Height].(*big.Int)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"indexerstore", "GetBlockHashes",
				height.Uint64(),
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getBlockHashes,
		},
	}
}

func BuildGetPositionByHashSubPlan() query.Step {
	getPosition := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			hash := ctx.Vars[QueryKey_TxHash].(evmCommon.Hash)
			txhashstr := string(hash.Bytes())

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"indexerstore", "GetPosition",
				txhashstr,
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getPosition,
		},
	}
}

func BuildGetBlockHeightByHashSubPlan() query.Step {
	getTxs := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			hash := ctx.Vars[QueryKey_BlockHash].(evmCommon.Hash)
			hashstr := string(hash.Bytes())

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"indexerstore", "GetHeightByHash",
				hashstr,
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	saveHeight := &query.ValueStep{
		Inner: getTxs,
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Height] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			height := ctx.Vars[QueryKey_Height].(*big.Int)
			if height.Cmp(big.NewInt(0)) < 0 {
				cont(nil, errors.New("not found"))
				return
			}
			cont(height, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getTxs,
			saveHeight,
			returnTx,
		},
	}
}

// receiptstore
func BuildGetReceiptsByHeightSubPlan() query.Step {
	getReceipts := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			height := ctx.Vars[QueryKey_Height].(*big.Int)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"receiptstore", "GetBlockReceipts",
				height.Uint64(),
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getReceipts,
		},
	}
}
func BuildGetReceiptByPositionSubPlan() query.Step {
	getReceipt := &query.RpcStep{
		Call: func(ctx *query.QueryContext, cont query.Continuation) {
			pos := ctx.Vars[QueryKey_Position].(*mstypes.Position)

			root := ctx.Root
			if root == nil {
				root = ctx
			}

			ctx.Ctx.InvokeRPC(
				"receiptstore", "Get",
				pos,
				"QueryContinuationAction",
				actor.NMeta("contId", root.RegisterCont(cont)),
			)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			getReceipt,
		},
	}
}
