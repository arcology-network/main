package scheduler

import (
	"context"
	"testing"

	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
)

type schdStoreMock struct {
}

func (mock *schdStoreMock) Load(ctx context.Context, _ *int, states *[]storage.SchdState) error {
	*states = []storage.SchdState{}
	return nil
}

func (mock *schdStoreMock) Save(ctx context.Context, state *storage.SchdState, _ *int) error {
	return nil
}

type executorMock struct {
	tb       testing.TB
	callTime int
	response []*mtypes.ExecutorResponses
}

func (mock *executorMock) ExecTxs(ctx context.Context, request *actor.Message, response *mtypes.ExecutorResponses) error {
	mock.tb.Log(request, request.Data.(*mtypes.ExecutorRequest))
	*response = *mock.response[mock.callTime]
	mock.callTime++
	return nil
}

func (mock *executorMock) GetConfig(ctx context.Context, _ *int, config *mtypes.ExecutorConfig) error {
	config.Concurrency = 4
	return nil
}

var (
	executor   executorMock
	arbitrator arbitratorMock
)

type arbitratorMock struct {
	tb       testing.TB
	callTime int
	response []*mtypes.ArbitratorResponse
}

func (mock *arbitratorMock) Arbitrate(ctx context.Context, request *actor.Message, response *mtypes.ArbitratorResponse) error {
	mock.tb.Log(request, request.Data.(*mtypes.ArbitratorRequest).TxsListGroup)
	*response = *mock.response[mock.callTime]
	mock.callTime++
	return nil
}
