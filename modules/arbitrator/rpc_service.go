package arbitrator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ctypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	kafkalib "github.com/arcology-network/component-lib/kafka/lib"
	"github.com/arcology-network/component-lib/log"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/main/modules/arbitrator/accumulator"
	arbitrator "github.com/arcology-network/main/modules/arbitrator/impl-arbitrator"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"go.uber.org/zap"
)

type RpcService struct {
	actor.WorkerThread
	wbs        *kafkalib.Waitobjs
	msgid      int64
	arbitrator *arbitrator.ArbitratorImpl
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
		rs.arbitrator = arbitrator.NewArbitratorImpl()
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
			euResults := v.Data.(*[]*types.ProcessedEuResult)
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

	var resultSelected *[]*types.ProcessedEuResult
	if results == nil {
		rs.AddLog(log.LogLevel_Error, "select euresults error")
		return errors.New("select euresults error")
	}

	if bValue, ok := results.(*[]*types.ProcessedEuResult); ok {
		resultSelected = bValue
	} else {
		rs.AddLog(log.LogLevel_Error, "select euresults type error")
		return errors.New("select euresults type error")
	}

	if resultSelected != nil && len(*resultSelected) > 0 {
		/*
			currentpath, err := common.GetCurrentDirectory()
			if err == nil {

				//fmt.Println("path:=" + currentpath)

				fmt.Printf("=============================height=%v\n", request.Height)

				path := currentpath + fmt.Sprintf("/%v", request.Height)
				if !common.DirExists(path) {
					os.Mkdir(path, os.ModePerm)
				}

				NumPerBatch := 500
				for i := 0; i < len(*resultSelected)/NumPerBatch; i++ {
					endIdx := (i + 1) * 500
					if endIdx > len(*resultSelected) {
						endIdx = len(*resultSelected)
					}
					tim, nums := rs.arbitrator.Insert(path+"/"+fmt.Sprintf("%v", i), (*resultSelected)[i*NumPerBatch:endIdx])
					rs.AddLog(log.LogLevel_Debug, "insert accessRecord***********", zap.Int("counts", nums), zap.Duration("time", tim))
				}

			}
		*/
		logid := rs.CheckPoint("Before detectConflict")
		interLog := rs.GetLogger(logid)
		conflictedList, left, right := detectConflict(rs.arbitrator, params.TxsListGroup, resultSelected, interLog)
		rs.CheckPoint("arbitrate return results***********", zap.Int("txResults", len(conflictedList)))

		response.ConflictedList = conflictedList
		response.CPairLeft = left
		response.CPairRight = right
		return nil
	}

	return nil
}

func detectConflict(arbitrator *arbitrator.ArbitratorImpl, txsListGroup [][]*ctypes.TxElement, euResults *[]*types.ProcessedEuResult, inlog *actor.WorkerThreadLogger) ([]*evmCommon.Hash, []uint32, []uint32) {
	timeDetails := make([]time.Duration, 10)
	// Make the results indexable.
	t := time.Now()
	euDict := make(map[evmCommon.Hash]*types.ProcessedEuResult, len(*euResults))
	for _, r := range *euResults {
		euDict[evmCommon.BytesToHash([]byte(r.Hash))] = r
	}
	timeDetails[0] = time.Since(t)
	// Prepare arguments for DetectConflict.
	t = time.Now()
	groups := make([][]*types.ProcessedEuResult, 0, len(txsListGroup))
	total := 0
	var maxBatch uint64 = 0
	for _, g := range txsListGroup {
		group := make([]*types.ProcessedEuResult, 0, len(g))
		total = total + len(g)
		for _, e := range g {
			group = append(group, euDict[*e.TxHash])
			if e.Batchid > maxBatch {
				maxBatch = e.Batchid
			}
		}
		groups = append(groups, group)
	}

	gpDict := make(map[uint32]int, total)
	for i, g := range txsListGroup {
		for _, e := range g {
			gpDict[e.Txid] = i
		}
	}
	timeDetails[1] = time.Since(t)
	// Arbitration.
	t = time.Now()
	inlog.Log(log.LogLevel_Debug, "----------------before arbitrator.DetectConflict", zap.Int("euDict", len(*euResults)))
	ids, _, flags, left, right, tims, txnums, details, totals := arbitrator.DetectConflict(groups)
	inlog.Log(log.LogLevel_Debug, "----------------after arbitrator.DetectConflict", zap.Durations("tims", tims), zap.Durations("details", details), zap.Int("totals", totals), zap.Int("txnums", txnums), zap.Int("conflictNums", len(ids)))
	timeDetails[2] = time.Since(t)
	// Unique conflict IDs.
	t = time.Now()
	uniqueConflicts := make(map[int]struct{})
	for i, conflict := range flags {
		if conflict {
			uniqueConflicts[gpDict[ids[i]]] = struct{}{}
		}
	}
	timeDetails[3] = time.Since(t)
	// Add conflicted groups into conflict list.
	t = time.Now()
	var conflictedList []*evmCommon.Hash
	for id := range uniqueConflicts {
		for _, e := range txsListGroup[id] {
			conflictedList = append(conflictedList, e.TxHash)
		}
	}
	timeDetails[4] = time.Since(t)
	inlog.Log(log.LogLevel_Debug, "Arbitration result", zap.Int("conflictNums", len(conflictedList)))
	// Make batch info indexable.
	t = time.Now()
	batches := make([][]*types.ProcessedEuResult, maxBatch+1)
	for i := range batches {
		batches[i] = make([]*types.ProcessedEuResult, 0, len(*euResults)/(i+1))
	}
	for i, g := range txsListGroup {
		if _, ok := uniqueConflicts[i]; ok {
			continue
		}
		for _, e := range g {
			if per, ok := euDict[*e.TxHash]; !ok {
				panic(fmt.Sprintf("tx hash not found, hash = %x, batch id = %v, tx id = %v\n", e.TxHash.Bytes(), e.Batchid, e.Txid))
			} else {
				batches[e.Batchid] = append(batches[e.Batchid], per)
			}
		}
	}
	timeDetails[5] = time.Since(t)
	// Prepare arguments for BatchCheck.
	t = time.Now()
	txs := make([]*types.ProcessedEuResult, 0, len(*euResults))
	for i := uint64(0); i <= maxBatch; i++ {
		txs = append(txs, batches[i]...)
	}
	timeDetails[6] = time.Since(t)
	// Accumulation.
	t = time.Now()
	results := accumulator.CheckBalance(batches)
	timeDetails[7] = time.Since(t)
	// Make results indexable.
	t = time.Now()
	conflictDict := make(map[evmCommon.Hash]struct{})
	for i, conflict := range results {
		if !conflict {
			continue
		}
		conflictDict[evmCommon.BytesToHash([]byte(txs[i].Hash))] = struct{}{}
	}
	timeDetails[8] = time.Since(t)
	// Add conflicted groups into conflict list.
	t = time.Now()
	for i, g := range txsListGroup {
		if _, ok := uniqueConflicts[i]; ok {
			continue
		}
		isConflict := false
		for _, e := range g {
			// If any of the member in this group is conflicted, the whole group is conflicted.
			if _, ok := conflictDict[*e.TxHash]; ok {
				isConflict = true
				break
			}
		}
		if isConflict {
			for _, e := range g {
				conflictedList = append(conflictedList, e.TxHash)
			}
		}
		//inlog.Log(log.LogLevel_Debug, "Accumulation result", zap.Int("conflictNums", len(conflictedList)))
	}
	timeDetails[9] = time.Since(t)
	inlog.Log(log.LogLevel_Debug, "detectConflict time details", zap.Durations("detail", timeDetails))
	return conflictedList, left, right
}
