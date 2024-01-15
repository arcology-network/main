package lib

/*
#cgo LDFLAGS: -L./ -lscheduler
#include "./scheduler.external.h"
*/
import "C"
import (
	"sync"
	"unsafe"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

var (
	nilAddress = evmCommon.Address{}
)

const (
	CALLEESIZE = 48
	BUFFERSIZE = 4 * 1024
)

type Scheduler struct {
	enginePtr unsafe.Pointer
}

var (
	singleton *Scheduler
	initErr   string
	initOnce  sync.Once
)

func (s *Scheduler) Stop() {
	C.Stop(s.enginePtr)
}

func Start(filename string) (*Scheduler, string) {
	initOnce.Do(func() {
		config := "{\"maxConflictRate\": 0.7, \"minParaBatch\" : 2, \"historyFile\" : \"\", \"debug\" : false}"

		configs := []byte(config)
		size := len(configs)

		buffer := make([]byte, BUFFERSIZE)
		c_log := (*C.char)(unsafe.Pointer(&buffer[0]))

		c_count := C.uint32_t(size)

		c_config := (*C.char)(unsafe.Pointer(&configs[0]))

		singleton = &Scheduler{}
		singleton.enginePtr = C.Start(c_config, c_count, c_log)
		initErr = ParseLog(buffer)
	})

	return singleton, initErr
}

func GetCallees(msgs []*types.StandardTransaction) ([]byte, []uint32) {
	callees := make([][]byte, len(msgs))
	txIds := make([]uint32, len(msgs))
	worker := func(start, end, idx int, args ...interface{}) {
		msgs := args[0].([]interface{})[0].([]*types.StandardTransaction)
		callees := args[0].([]interface{})[1].([][]byte)
		txIds := args[0].([]interface{})[2].([]uint32)
		for i := start; i < end; i++ {
			to := msgs[i].NativeMessage.To
			if to == nil {
				callees[i] = []byte(nilAddress.Hex()[2:])
			} else {
				callees[i] = []byte(msgs[i].NativeMessage.To.Hex()[2:])
			}
			pads := make([]byte, CALLEESIZE-len(callees[i]))
			callees[i] = append(callees[i], pads...)

			txIds[i] = uint32(i)
		}
	}
	common.ParallelWorker(len(msgs), 4, worker, msgs, callees, txIds)
	// for _, callee := range callees {
	// 	fmt.Printf("calle=%v\n", callee)
	// }
	return codec.Byteset(callees).Flatten(), txIds
}

type Item struct {
	id     int
	branch int
}

func ParseResult(msgs []*types.StandardTransaction, orderIDs []uint32, branches []uint32, generations []uint32) [][]*types.ExecutingSequence {
	count := len(msgs)
	idx_id := make([]int, count)
	idx_branche := make([]int, count)
	idx_generation := make([]int, count)

	worker := func(start, end, idx int, args ...interface{}) {
		orderIDs := args[0].([]interface{})[0].([]uint32)
		branches := args[0].([]interface{})[1].([]uint32)
		generations := args[0].([]interface{})[2].([]uint32)
		idx_id := args[0].([]interface{})[3].([]int)
		idx_branche := args[0].([]interface{})[4].([]int)
		idx_generation := args[0].([]interface{})[5].([]int)
		for i := start; i < end; i++ {
			idx_id[i] = int(orderIDs[i])
			idx_branche[i] = int(branches[i])
			idx_generation[i] = int(generations[i])
		}
	}
	common.ParallelWorker(count, 4, worker, orderIDs, branches, generations, idx_id, idx_branche, idx_generation)

	mapitems := map[int]*[]*Item{}
	for i := range idx_id {
		item := Item{
			id:     idx_id[i],
			branch: idx_branche[i],
		}

		if list, ok := mapitems[idx_generation[i]]; ok {
			(*list) = append((*list), &item)
		} else {
			list := []*Item{&item}
			mapitems[idx_generation[i]] = &list
		}
	}

	items := make([][]*Item, len(mapitems))
	for i, list := range mapitems {
		items[i] = *list
	}
	idxes := make([][][]int, len(items))
	for i, its := range items {
		mapids := map[int]*[]int{}
		for _, it := range its {
			if list, ok := mapids[it.branch]; ok {
				(*list) = append((*list), it.id)
			} else {
				list := []int{it.id}
				mapids[it.branch] = &list
			}
		}
		ids := make([][]int, len(mapids))
		for key, list := range mapids {
			ids[key] = *list
		}
		idxes[i] = ids
	}

	sequences := make([][]*types.ExecutingSequence, len(idxes))
	for i, list := range idxes {
		executingSequenceList := make([]*types.ExecutingSequence, 0, len(list))
		parallels := make([]*types.StandardTransaction, 0, len(msgs))
		for _, ids := range list {
			if len(ids) == 1 {
				parallels = append(parallels, msgs[ids[0]])
				continue
			}
			seqMsgs := make([]*types.StandardTransaction, len(ids))
			for k, id := range ids {
				seqMsgs[k] = msgs[id]
			}
			executingSequenceList = append(executingSequenceList, types.NewExecutingSequence(seqMsgs, false))
		}
		if len(parallels) > 0 {
			executingSequenceList = append(executingSequenceList, types.NewExecutingSequence(parallels, true))
		}
		sequences[i] = executingSequenceList
	}
	return sequences
}

func ParseLog(buffer []byte) string {
	var rstsize int
	for i, c := range buffer {
		if c == 0 {
			rstsize = i
			break
		}
	}
	return string(buffer[:rstsize])
}
func (s *Scheduler) Schedule(msgs []*types.StandardTransaction, height uint64) ([][]*types.ExecutingSequence, string) {
	if len(msgs) == 0 {
		return [][]*types.ExecutingSequence{}, ""
	}
	buffer := make([]byte, BUFFERSIZE)
	c_log := (*C.char)(unsafe.Pointer(&buffer[0]))

	count := len(msgs)
	c_count := C.uint32_t(count)

	callees, txids := GetCallees(msgs)

	// fmt.Printf("callees=%v\n", callees)
	// fmt.Printf("txids=%v\n", txids)

	c_callees := (*C.char)(unsafe.Pointer(&callees[0]))

	c_txids := (*C.uint32_t)(unsafe.Pointer(&txids[0]))

	newTxOrder := make([]uint32, count)
	branches := make([]uint32, count)
	generations := make([]uint32, count)

	c_newTxOrder := (*C.uint32_t)(unsafe.Pointer(&newTxOrder[0]))
	c_branches := (*C.uint32_t)(unsafe.Pointer(&branches[0]))
	c_generations := (*C.uint32_t)(unsafe.Pointer(&generations[0]))
	C.Schedule(s.enginePtr, c_callees, c_txids, c_newTxOrder, c_branches, c_generations, c_count, c_log)
	return ParseResult(msgs, newTxOrder, branches, generations), ParseLog(buffer)
}

func PaddingRight(ones []*evmCommon.Address, twos []*evmCommon.Address) ([]byte, []byte) {
	padOnes := make([][]byte, len(ones))
	padTwos := make([][]byte, len(twos))

	pads := make([]byte, CALLEESIZE-len(ones[0].Hex()[2:]))
	// fmt.Printf("pads=%v\n", pads)
	for i, addr := range ones {
		// fmt.Printf("addr.Hex()=%v\n", addr.Hex()[2:])
		addrData := []byte(addr.Hex()[2:])
		padOnes[i] = append(padOnes[i], addrData...)
		padOnes[i] = append(padOnes[i], pads...)
	}

	for i, addr := range twos {
		addrData := []byte(addr.Hex()[2:])
		padTwos[i] = append(padTwos[i], addrData...)
		padTwos[i] = append(padTwos[i], pads...)
	}
	codec.Byteset(padOnes).Flatten()

	return codec.Byteset(padOnes).Flatten(), codec.Byteset(padTwos).Flatten()
}

func (s *Scheduler) Update(ones []*evmCommon.Address, twos []*evmCommon.Address) string {
	buffer := make([]byte, BUFFERSIZE)
	c_log := (*C.char)(unsafe.Pointer(&buffer[0]))

	count := len(ones)
	c_count := C.uint32_t(count)

	paddedOnes, paddedTwos := PaddingRight(ones, twos)

	c_paddedOnes := (*C.char)(unsafe.Pointer(&paddedOnes[0]))
	c_paddedTwos := (*C.char)(unsafe.Pointer(&paddedTwos[0]))

	C.UpdateHistory(s.enginePtr, c_paddedOnes, c_paddedTwos, nil, c_count, c_log)

	return ParseLog(buffer)
}
