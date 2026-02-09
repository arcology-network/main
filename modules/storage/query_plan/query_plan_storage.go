package queryplan

import (
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type StorageQueryPlan struct {
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
			ctx.Vars[QueryKey_AccountAddress] = fmt.Sprintf("%x", request.Address.Bytes())
			ctx.Vars[QueryKey_StorageKey] = request.Key
			cont(nil, nil)
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
			getStorage,
			returnTx,
		},
	}

}
