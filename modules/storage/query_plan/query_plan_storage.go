package queryplan

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type StorageQueryPlan struct {
	GetLastHeight func() uint64
}

func (p *StorageQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}

func (p *StorageQueryPlan) buildSteps() query.Step {

	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			request := ctx.Req.(*mtypes.RequestStorage)
			ctx.Vars[QueryKey_StorageKey] = request.Key
			ctx.Vars[QueryKey_Address] = request.Address
			ctx.Vars[QueryKey_BlockParams] = request.BlockParams
			cont(nil, nil)
		},
	}
	getHeight := &query.CallStep{
		Plan: BuildGetHeightByHashOrNumberSubPlan(p.GetLastHeight),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Height] = v
		},
	}

	getState := &query.CallStep{
		Plan: BuildGetStateRootByHeightSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_StateRoot] = v
		},
	}
	getStorage := &query.CallStep{
		Plan: BuildGetStorageSubPlan(),
		Bind: func(ctx *query.QueryContext, v any) {
			ctx.Vars[QueryKey_Storage] = v
		},
	}

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			storage := ctx.Vars[QueryKey_Storage]
			cont(storage, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getHeight,
			getState,
			getStorage,
			returnTx,
		},
	}

}
