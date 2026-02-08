package query

type QueryPlan interface {
	Start(ctx *QueryContext, cont Continuation)
}

type SimplePlan struct {
	Root Step
}

func NewSimplePlan(root Step) *SimplePlan {
	return &SimplePlan{Root: root}
}

func (p *SimplePlan) Start(ctx *QueryContext, cont Continuation) {
	StartStep(ctx, p.Root, cont)
}
