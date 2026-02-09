package queryplan

import (
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

func FuncEnsure(cond func(*query.QueryContext) bool, err error) query.Step {
	return &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			if cond(ctx) {
				cont(nil, err)
				return
			}
			cont(nil, nil)
		},
	}
}

func BuildGetHeightByHashOrNumberSubPlan(GetLastHeight func() uint64) query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			blockParams := ctx.Vars[QueryKey_BlockParams].(*mtypes.BlockNumberOrHash)
			ctx.Vars[QueryKey_BlockHashExist] = false
			if hash, ok := blockParams.Hash(); ok {
				ctx.Vars[QueryKey_BlockHash] = hash
				ctx.Vars[QueryKey_BlockHashExist] = true
			} else if height, ok := blockParams.Number(); ok {
				ctx.Vars[QueryKey_Height] = height
			}
			cont(nil, nil)
		},
	}
	getHeight := &query.IfStep{
		Cond: func(ctx *query.QueryContext) bool {
			return ctx.Vars[QueryKey_BlockHashExist].(bool)
		},
		Then: BuildGetBlockHeightByHashSubPlan(),
		Else: &query.FuncStep{
			Do: func(ctx *query.QueryContext, cont query.Continuation) {
				cont(ctx.Vars[QueryKey_Height], nil)
			},
		},
	}

	saveHeight := &query.ValueStep{
		Inner: getHeight,
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Height] = v
		},
	}

	ensureHeight := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			height := ctx.Vars[QueryKey_Height].(*big.Int)
			blockParams := ctx.Vars[QueryKey_BlockParams].(*mtypes.BlockNumberOrHash)

			finalHeight, err := blockParams.ParseBlockNumber(height, GetLastHeight())
			if err != nil {
				cont(nil, err)
				return
			}
			ctx.Vars[QueryKey_Height] = finalHeight
			cont(nil, nil)
		},
	}

	FinalReturn := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			fh := ctx.Vars[QueryKey_Height]
			cont(fh, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			saveHeight,
			ensureHeight,
			FinalReturn,
		},
	}
}
