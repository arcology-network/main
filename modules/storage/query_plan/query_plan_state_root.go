package queryplan

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/query"
)

type StateRootQueryPlan struct {
	GetLastHeight func() uint64
}

func (p *StateRootQueryPlan) Start(
	ctx *query.QueryContext,
	cont query.Continuation,
) {
	root := p.buildSteps()
	query.StartStep(ctx, root, cont)
}

func (p *StateRootQueryPlan) buildSteps() query.Step {
	params := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			params := ctx.Req.(*mtypes.StateRootRequest)
			ctx.Vars[QueryKey_BlockParams] = params.BlockParam
			ctx.Vars[QueryKey_RequestId] = params.ReqId
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

	returnTx := &query.FuncStep{
		Do: func(ctx *query.QueryContext, cont query.Continuation) {
			stateRoot := ctx.Vars[QueryKey_StateRoot]
			reqId := ctx.Vars[QueryKey_RequestId]
			cont(&mtypes.StateRootResponse{
				Root:  stateRoot.([]byte),
				ReqId: reqId.(string),
			}, nil)
		},
	}

	return &query.SequentialStep{
		Steps: []query.Step{
			params,
			getHeight,
			getState,
			returnTx,
		},
	}
}
