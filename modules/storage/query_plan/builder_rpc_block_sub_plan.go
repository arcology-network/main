package queryplan

import (
	"fmt"
	"math/big"

	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type RpcBlockResult struct {
	Block  *mtypes.RPCBlock
	Signer uint8
}

func BuildGetRpcBlockSubPlan() query.Step {
	validateHeight := &query.ReturnFromSubPlanStep{
		Cond: func(ctx *query.QueryContext) bool {
			h := ctx.Vars[QueryKey_Height]
			return h == nil || h.(*big.Int).Sign() < 0
		},
		Value: func(ctx *query.QueryContext) any {
			return &RpcBlockResult{
				Block:  &mtypes.RPCBlock{},
				Signer: 0,
			}
		},
	}

	getBlock := &query.CallStep{
		Plan: BuildGetBlockByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_MonacoBlock] = v
		},
	}

	ensureBlock := &query.ReturnFromSubPlanStep{
		Cond: func(ctx *query.QueryContext) bool {
			return ctx.Vars[QueryKey_MonacoBlock] == nil
		},
		Value: func(ctx *query.QueryContext) any {
			return &RpcBlockResult{
				Block:  &mtypes.RPCBlock{},
				Signer: 0,
			}
		},
	}

	buildHeader := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			block := ctx.Vars[QueryKey_MonacoBlock].(*mtypes.MonacoBlock)
			ctx.Vars[QueryKey_BlockHash] = evmCommon.BytesToHash(block.Blockhash)
			var header evmTypes.Header
			for i := range block.Headers {
				if block.Headers[i][0] != mtypes.AppType_Eth {
					continue
				}
				if err := header.UnmarshalJSON(block.Headers[i][1:]); err != nil {
					cont(nil, err)
					return
				}
			}

			data, err := block.GobEncode()
			if err != nil {
				cont(nil, err)
				return
			}

			ctx.Vars[QueryKey_RpcBlock] = &mtypes.RPCBlock{
				Size:   uint64(len(data)),
				Header: &header,
			}
			ctx.Vars[QueryKey_SignerType] = block.Signer
			cont(nil, nil)
		},
	}

	onlyHeaderReturn := &query.ReturnFromSubPlanStep{
		Cond: func(ctx *query.QueryContext) bool {
			return ctx.Vars[QueryKey_OnlyHeader].(bool)
		},
		Value: func(ctx *query.QueryContext) any {
			return &RpcBlockResult{
				Block:  ctx.Vars[QueryKey_RpcBlock].(*mtypes.RPCBlock),
				Signer: ctx.Vars[QueryKey_SignerType].(uint8),
			}
		},
	}

	fillFullTx := &query.IfStep{
		Cond: func(ctx *query.QueryContext) bool {
			return ctx.Vars[QueryKey_Fulltx].(bool)
		},
		Then: BuildFillFullTxStepSubPlan(),
	}

	fillHashes := &query.IfStep{
		Cond: func(ctx *query.QueryContext) bool {
			return !ctx.Vars[QueryKey_Fulltx].(bool)
		},
		Then: BuildFillHashesSubPlan(),
	}

	finalReturn := &query.ReturnFromSubPlanStep{
		Cond: nil,
		Value: func(ctx *query.QueryContext) any {
			return &RpcBlockResult{
				Block:  ctx.Vars[QueryKey_RpcBlock].(*mtypes.RPCBlock),
				Signer: ctx.Vars[QueryKey_SignerType].(uint8),
			}
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			validateHeight,
			getBlock,
			ensureBlock,
			buildHeader,
			onlyHeaderReturn,
			fillFullTx,
			fillHashes,
			finalReturn,
		},
	}
}

func BuildFillFullTxStepSubPlan() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			mblock := ctx.Vars[QueryKey_MonacoBlock].(*mtypes.MonacoBlock)
			txs := make([]interface{}, 0, len(mblock.Txs))
			ctx.Vars[QueryKey_Transactions] = txs
			cont(nil, nil)
		},
	}

	buildTransactions := &query.ForEachStep{
		Items: func(ctx *query.QueryContext) []interface{} {
			mblock := ctx.Vars[QueryKey_MonacoBlock].(*mtypes.MonacoBlock)
			items := make([]interface{}, len(mblock.Txs))
			for i := range mblock.Txs {
				items[i] = i
			}
			return items
		},
		Body: func(item any) query.Step {
			return &query.SequentialStep{
				Steps: []query.Step{

					&query.FuncStep{
						Do: func(ctx *query.QueryContext, cont query.Continuation) {
							height := ctx.Vars[QueryKey_Height].(*big.Int)
							ctx.Vars[QueryKey_Position] = &mstypes.Position{
								Height:     height.Uint64(),
								IdxInBlock: item.(int),
							}
							cont(nil, nil)
						},
					},

					&query.CallStep{
						Plan: BuildGetTransactionByPositionSubPlan(),
						Bind: func(ctx *query.QueryContext, v any) {
							ctx.Vars[QueryKey_RpcTransaction] = v
						},
					},
					&query.ReturnFromSubPlanStep{
						Value: func(ctx *query.QueryContext) any {
							return ctx.Vars[QueryKey_RpcTransaction]
						},
					},
				},
			}
		},
		Collect: func(parent *query.QueryContext, _ any, sub *query.QueryContext) {
			fmt.Printf("***************BuildFillFullTxStepSubPlan.buildTransactions.Body item done*********\n")
			txs := parent.Vars[QueryKey_Transactions].([]interface{})
			txs = append(txs, sub.Vars[QueryKey_RpcTransaction])
			parent.Vars[QueryKey_Transactions] = txs
		},
	}
	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			ts := ctx.Vars[QueryKey_Transactions].([]interface{})
			block := ctx.Vars[QueryKey_RpcBlock].(*mtypes.RPCBlock)
			block.Transactions = ts
			cont(block, nil)
		},
	}
	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			buildTransactions,
			finalReturn,
		},
	}
}
func BuildFillHashesSubPlan() query.Step {
	getHashess := &query.CallStep{
		Plan: BuildGetBlockHashesByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Hashes] = v
		},
	}
	finalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			hashes := ctx.Vars[QueryKey_Hashes].([]string)
			block := ctx.Vars[QueryKey_RpcBlock].(*mtypes.RPCBlock)
			block.Transactions = convertHashes(hashes)
			cont(block, nil)
		},
	}
	return &query.SequentialStep{
		Steps: []query.Step{
			getHashess,
			finalReturn,
		},
	}
}

func convertHashes(hashstr []string) []interface{} {
	hashes := make([]interface{}, len(hashstr))
	for i := range hashstr {
		hashes[i] = evmCommon.HexToHash(hashstr[i])
	}
	return hashes
}
