package queryplan

import (
	"github.com/arcology-network/streamer/query"
)

type InlineQueryPlan struct {
	Fn func(*query.QueryContext) (interface{}, error)
}

func (p *InlineQueryPlan) Start(ctx *query.QueryContext, cont query.Continuation) {
	resp, err := p.Fn(ctx)
	cont(resp, err)
}
