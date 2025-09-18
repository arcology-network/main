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

package arbitrator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/arcology-network/common-lib/exp/slice"
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/main/modules/arbitrator/types"
	mtypes "github.com/arcology-network/main/types"
	arbitratorn "github.com/arcology-network/scheduler/arbitrator"
	"github.com/arcology-network/storage-committer/type/univalue"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"
	"github.com/arcology-network/streamer/actor"
	kafkalib "github.com/arcology-network/streamer/kafka/lib"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
)

type RpcService struct {
	actor.WorkerThread
	wbs        *kafkalib.Waitobjs
	msgid      int64
	arbitrator *arbitratorn.Arbitrator
}

var (
	rpcServiceSingleton actor.IWorkerEx
	initOnce            sync.Once
)

// return a Subscriber struct
func NewRpcService(lanes int, groupid string) actor.IWorkerEx {
	initOnce.Do(func() {
		rs := RpcService{}
		rs.Set(lanes, groupid)
		rs.msgid = 0
		rpcServiceSingleton = &rs
		rs.arbitrator = arbitratorn.NewArbitrator()
	})
	return rpcServiceSingleton
}

func (rs *RpcService) Inputs() ([]string, bool) {
	return []string{actor.MsgEuResultSelected, actor.MsgPreProcessedImportEuResults}, false
}

func (rs *RpcService) Outputs() map[string]int {
	return map[string]int{
		actor.MsgArbitrateReapinglist:  1,
		actor.MsgPreProcessedEuResults: 1,
	}
}

func (rs *RpcService) OnStart() {
	rs.wbs = kafkalib.StartWaitObjects()
}

func (rs *RpcService) OnMessageArrived(msgs []*actor.Message) error {

	for _, v := range msgs {
		switch v.Name {
		case actor.MsgEuResultSelected:
			euResults := v.Data.(*[]*types.AccessRecord)
			rs.AddLog(log.LogLevel_Debug, "received selectedEuresult***********", zap.Int64("msgid", rs.msgid))
			rs.wbs.Update(rs.msgid, euResults)

			fmt.Printf("height=%v\n", v.Height)
		case actor.MsgPreProcessedImportEuResults:
			ars := v.Data.([]*types.AccessRecord)
			newTrans := make([]*univalue.Univalue, 0, len(ars)*50)
			for i := range ars {
				newTrans = append(newTrans, ars[i].Accesses...)
			}
			rs.arbitrator.Insert(newTrans)

			rs.MsgBroker.Send(actor.MsgPreProcessedEuResults, ars)
		}
	}

	return nil
}

func (rs *RpcService) Arbitrate(ctx context.Context, request *actor.Message, response *mtypes.ArbitratorResponse) error {
	lstMessage := request.CopyHeader()
	rs.ChangeEnvironment(lstMessage)
	params := request.Data.(*mtypes.ArbitratorRequest)

	reapinglist := ctypes.ReapingList{
		List: slice.Flatten(params.TxsListGroup),
	}

	rs.msgid = rs.msgid + 1
	rs.CheckPoint("start arbitrate request***********", zap.Int("txs", len(reapinglist.List)))
	rs.wbs.AddWaiter(rs.msgid)
	rs.MsgBroker.Send(actor.MsgArbitrateReapinglist, &reapinglist)

	rs.wbs.Waitforever(rs.msgid)
	results := rs.wbs.GetData(rs.msgid)

	var resultSelected *[]*types.AccessRecord
	if results == nil {
		rs.AddLog(log.LogLevel_Error, "select euresults error")
		return errors.New("select euresults error")
	}

	if bValue, ok := results.(*[]*types.AccessRecord); ok {
		resultSelected = bValue
	} else {
		rs.AddLog(log.LogLevel_Error, "select euresults type error")
		return errors.New("select euresults type error")
	}

	if len(params.TxsListGroup) > 1 && resultSelected != nil && len(*resultSelected) > 0 {
		rs.CheckPoint("Before detectConflict", zap.Int("tx nums", len(*resultSelected)))

		conflicts := rs.arbitrator.Detect()
		fmt.Printf("----------arbitrate result-------conflicts Info------------\n")
		arbitratorn.Conflicts(conflicts).Print()
		response.CPairLeft, response.CPairRight = parseResult(conflicts)
		rs.CheckPoint("arbitrate return results***********", zap.Uint64s("left", response.CPairLeft), zap.Uint64s("right", response.CPairRight))
		// return nil
	}
	rs.arbitrator.Clear()
	return nil
}

func parseRequests(txsListGroup [][]evmCommon.Hash, results *[]*types.AccessRecord) ([][]uint64, [][]*univaluepk.Univalue) {
	mp := map[[32]byte]*types.AccessRecord{}
	for _, result := range *results {
		mp[result.TxHash] = result
	}
	groupIDs := make([][]uint64, len(txsListGroup))
	records := make([][]*univaluepk.Univalue, len(txsListGroup))
	for i, row := range txsListGroup {
		ids := make([]uint64, 0, len(row))
		transactations := []*univaluepk.Univalue{}
		for _, e := range row {
			result := mp[[32]byte(e.Bytes())]
			ids = append(ids, slice.Fill(make([]uint64, len(result.Accesses)), uint64(i))...)
			transactations = append(transactations, result.Accesses...)
		}
		groupIDs[i] = ids
		records[i] = transactations
	}
	return groupIDs, records
}

func parseResult(conflits arbitratorn.Conflicts) ([]uint64, []uint64) {
	_, _, pairs := conflits.ToDict()
	left := make([]uint64, 0, len(pairs))
	right := make([]uint64, 0, len(pairs))
	for _, pair := range pairs {
		left = append(left, pair[0])
		right = append(right, pair[1])
	}
	return left, right
}
