/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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
