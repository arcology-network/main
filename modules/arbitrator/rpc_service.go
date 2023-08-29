package arbitrator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/arcology-network/common-lib/common"
	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	kafkalib "github.com/arcology-network/component-lib/kafka/lib"
	"github.com/arcology-network/component-lib/log"
	arbitratorn "github.com/arcology-network/concurrenturl/arbitrator"
	"github.com/arcology-network/concurrenturl/interfaces"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"github.com/arcology-network/vm-adaptor/execution"
	"go.uber.org/zap"
)

type RpcService struct {
	actor.WorkerThread
	wbs   *kafkalib.Waitobjs
	msgid int64
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
	})
	return rpcServiceSingleton
}

func (rs *RpcService) Inputs() ([]string, bool) {
	return []string{actor.MsgEuResultSelected}, false
}

func (rs *RpcService) Outputs() map[string]int {
	return map[string]int{
		actor.MsgArbitrateReapinglist: 1,
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
		}
	}

	return nil
}

func (rs *RpcService) Arbitrate(ctx context.Context, request *actor.Message, response *ctypes.ArbitratorResponse) error {
	lstMessage := request.CopyHeader()
	rs.ChangeEnvironment(lstMessage)
	params := request.Data.(*ctypes.ArbitratorRequest)
	list := []*evmCommon.Hash{}
	for _, rows := range params.TxsListGroup {
		for _, element := range rows {
			list = append(list, element.TxHash)
		}
	}
	reapinglist := ctypes.ReapingList{
		List: list,
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

	if resultSelected != nil && len(*resultSelected) > 0 {
		rs.CheckPoint("Before detectConflict", zap.Int("tx nums", len(*resultSelected)))

		gen := execution.NewGeneration(0, 0, nil)
		conflicts := gen.Detect(parseRequests(params.TxsListGroup, resultSelected))
		// conflicts.Print()
		response.ConflictedList, response.CPairLeft, response.CPairRight = parseResult(params.TxsListGroup, conflicts)
		rs.CheckPoint("arbitrate return results***********", zap.Int("ConflictedList", len(response.ConflictedList)), zap.Int("left", len(response.CPairLeft)), zap.Int("right", len(response.CPairRight)))
		return nil
	}

	return nil
}

func parseRequests(txsListGroup [][]*ctypes.TxElement, results *[]*types.AccessRecord) ([][]uint32, [][]interfaces.Univalue) {
	mp := map[[32]byte]*types.AccessRecord{}
	for _, result := range *results {
		mp[result.TxHash] = result
	}
	groupIDs := make([][]uint32, len(txsListGroup))
	records := make([][]interfaces.Univalue, len(txsListGroup))
	for i, row := range txsListGroup {
		ids := make([]uint32, 0, len(row))
		transactations := []interfaces.Univalue{}
		for _, e := range row {
			result := mp[[32]byte(e.TxHash.Bytes())]
			ids = append(ids, common.Fill(make([]uint32, len(result.Accesses)), uint32(i))...)
			transactations = append(transactations, result.Accesses...)
		}
		groupIDs[i] = ids
		records[i] = transactations
	}
	return groupIDs, records
}

func parseResult(txsListGroup [][]*ctypes.TxElement, conflits arbitratorn.Conflicts) ([]*evmCommon.Hash, []uint32, []uint32) {
	dic := map[uint32]*evmCommon.Hash{}
	for _, row := range txsListGroup {
		for _, e := range row {
			dic[e.Txid] = e.TxHash
		}
	}
	idsmp, _, paris := conflits.ToDict()
	confiltList := []*evmCommon.Hash{}
	for _, id := range common.MapKeys(*idsmp) {
		confiltList = append(confiltList, dic[id])
	}
	left := make([]uint32, 0, len(paris))
	right := make([]uint32, 0, len(paris))
	for _, par := range paris {
		left = append(left, par[0])
		right = append(right, par[1])
	}
	return confiltList, left, right
}
