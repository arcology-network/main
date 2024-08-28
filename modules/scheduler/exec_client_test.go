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
	"math/big"
	"reflect"
	"testing"
	"time"

	eucommon "github.com/arcology-network/common-lib/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	brokerpk "github.com/arcology-network/streamer/broker"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/log"
)

type execRpcMock struct {
	id          string
	concurrency int
	tb          testing.TB
	calls       [][]int
}

func newExecRpcMock(id string, concurrency int, tb testing.TB) *execRpcMock {
	return &execRpcMock{
		id:          id,
		concurrency: concurrency,
		tb:          tb,
		calls:       [][]int{},
	}
}

func (mock *execRpcMock) check(expected [][]int) {
	if !reflect.DeepEqual(expected, mock.calls) {
		mock.tb.Errorf("check failed, expected: %v, got: %v\n", expected, mock.calls)
	}
}

func (mock *execRpcMock) ExecTxs(_ context.Context, request *actor.Message, response *mtypes.ExecutorResponses) error {
	req := request.Data.(*mtypes.ExecutorRequest)
	numMsgs := 0
	for _, seq := range req.Sequences {
		numMsgs += len(seq.Msgs)
	}
	time.Sleep(time.Duration(numMsgs) * time.Millisecond)

	mock.calls = append(mock.calls, []int{len(req.Sequences), numMsgs})
	mock.tb.Logf("[%s] num of sequence: %d, num of msgs: %d\n", mock.id, len(req.Sequences), numMsgs)
	return nil
}

func (mock *execRpcMock) GetConfig(_ context.Context, _ *int, config *mtypes.ExecutorConfig) error {
	config.Concurrency = mock.concurrency
	return nil
}

type workerMock struct {
	actor.WorkerThread
}

func TestExecClientMakeRequestBasic(t *testing.T) {
	runExecClientTestCase(
		t,
		[]string{"executor-1"},
		[]int{4, 4},
		550,
		4,
		[][][]int{
			{
				{4, 400},
				{2, 150},
			},
		},
	)
}

func TestExecClientParallelismControlTest1(t *testing.T) {
	runExecClientTestCase(
		t,
		[]string{"executor-1", "executor-2"},
		[]int{4, 4},
		1350,
		4,
		[][][]int{
			{
				{4, 400},
				{4, 400},
				{4, 400},
				{2, 150},
			},
			{},
		},
	)
}

func TestExecClientParallelismControlTest2(t *testing.T) {
	runExecClientTestCase(
		t,
		[]string{"executor-1", "executor-2"},
		[]int{3, 3},
		1000,
		5,
		[][][]int{
			{
				{3, 300},
				{3, 300},
			},
			{
				{2, 200},
				{2, 200},
			},
		},
	)
}

func runExecClientTestCase(
	tb testing.TB,
	executors []string,
	concurrencies []int,
	numMsgs int,
	parallelism int,
	expected [][][]int,
) {
	log.InitLog("exec-client-test.log", "./log.toml", "tester", "tester", 0)
	intf.RPCCreator = func(serviceAddr, basepath string, zkAddrs []string, rcvrs, fns []interface{}) {}

	execs := make([]*execRpcMock, len(executors))
	for i := range executors {
		execs[i] = newExecRpcMock(executors[i], concurrencies[i], tb)
		intf.Router.Register(executors[i], execs[i], "rpc-server-addr", "zk-server-addr")
	}
	intf.Router.SetAvailableServices(executors)

	worker := &workerMock{}
	worker.Init("mock worker", brokerpk.NewStatefulStreamer())

	msgs := make([]*eucommon.StandardMessage, numMsgs)
	for i := range msgs {
		msgs[i] = &eucommon.StandardMessage{}
	}
	// ids := make([]uint32, len(msgs))
	client := NewExecClient(executors, 100)
	client.Run(
		// map[evmCommon.Hash]*schtyp.Message{
		// 	{}: {
		// 		Precedings: &[]*evmCommon.Hash{},
		// 	},
		// },
		[]*mtypes.ExecutingSequence{
			{
				Msgs:     msgs,
				Parallel: true,
				// Txids:    ids,
			},
		},
		new(big.Int),
		&actor.Message{},
		worker.GetLogger(0),
		uint64(1),
		parallelism,
		1,
	)

	for i := range executors {
		execs[i].check(expected[i])
	}
}
