package query

import "fmt"

type StepObserver interface {
	OnStepStart(ctx *QueryContext, step Step)
	OnStepFinish(ctx *QueryContext, step Step, err error)

	OnRPCSend(ctx *QueryContext, step Step, req interface{})
	OnRPCRecv(ctx *QueryContext, step Step, resp interface{}, err error)
}

func stepName(step Step) string {
	return fmt.Sprintf("%T", step)
}

type MockObserver struct {
	stepsStarted  []string
	stepsFinished []string
}

func (o *MockObserver) OnStepStart(ctx *QueryContext, step Step) {
	o.stepsStarted = append(o.stepsStarted, stepName(step))
}

func (o *MockObserver) OnStepFinish(ctx *QueryContext, step Step, err error) {
	o.stepsFinished = append(o.stepsFinished, stepName(step))
}

func (o *MockObserver) OnRPCSend(ctx *QueryContext, step Step, req interface{}) {}
func (o *MockObserver) OnRPCRecv(ctx *QueryContext, step Step, resp interface{}, err error) {
}
