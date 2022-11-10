package arbitrator

import (
	"sync"
	"time"
	"unsafe"

	"github.com/arcology-network/main/modules/arbitrator/types"
	"github.com/arcology-network/urlarbitrator-engine/go-wrapper"
)

type ArbitratorImpl struct {
	arbitrator unsafe.Pointer

	Txs       []uint32
	Paths     []string
	Reads     []uint32
	Writes    []uint32
	Composite []bool
}

var (
	arbitratorSingleton *ArbitratorImpl
	initOnce            sync.Once
)

func NewArbitratorImpl() *ArbitratorImpl {
	preAllocSize := 5000000
	initOnce.Do(func() {
		arbitratorSingleton = &ArbitratorImpl{
			arbitrator: wrapper.Start(),
			Txs:        make([]uint32, 0, preAllocSize),
			Paths:      make([]string, 0, preAllocSize),
			Reads:      make([]uint32, 0, preAllocSize),
			Writes:     make([]uint32, 0, preAllocSize),
			Composite:  make([]bool, 0, preAllocSize),
		}
	})
	return arbitratorSingleton
}

func (arb *ArbitratorImpl) Clear() {
	wrapper.Clear(arb.arbitrator)
	//arb.arbitrator = wrapper.Start()
	arb.Reset()
}
func (arb *ArbitratorImpl) Reset() {
	arb.Txs = arb.Txs[:0]
	arb.Paths = arb.Paths[:0]
	arb.Reads = arb.Reads[:0]
	arb.Writes = arb.Writes[:0]
	arb.Composite = arb.Composite[:0]
}
func (arb *ArbitratorImpl) Insert(groups []*types.ProcessedEuResult) ([]time.Duration, int) {
	//func (arb *ArbitratorImpl) Insert(filename string, groups []*types.ProcessedEuResult) (time.Duration, int) {
	arb.Reset()
	for _, per := range groups {
		if per == nil {
			continue
		}
		arb.Txs = append(arb.Txs, per.Txs...)
		arb.Paths = append(arb.Paths, per.Paths...)
		arb.Reads = append(arb.Reads, per.Reads...)
		arb.Writes = append(arb.Writes, per.Writes...)
		arb.Composite = append(arb.Composite, per.Composite...)
	}

	/**************************************************************************************************/
	//filename := "testdata"
	// common.AppendToFile(filename, "=============================================")
	// common.AddToLogFile(filename, "txs", arb.Txs)
	// common.AddToLogFile(filename, "paths", arb.Paths)
	// common.AddToLogFile(filename, "reads", arb.Reads)
	// common.AddToLogFile(filename, "writes", arb.Writes)
	// common.AddToLogFile(filename, "composite", arb.Composite)
	/**************************************************************************************************/

	tim := wrapper.Insert(arb.arbitrator, arb.Txs, arb.Paths, arb.Reads, arb.Writes, arb.Composite)

	return tim, len(arb.Txs)
}

func (arb *ArbitratorImpl) DetectConflict(groups [][]*types.ProcessedEuResult) ([]uint32, []uint32, []bool, []uint32, []uint32, []time.Duration, int, []time.Duration, int) {
	tims := make([]time.Duration, 3)
	begintime := time.Now()
	total := 0
	totalTransitions := 0
	whitelist := make([][]uint32, len(groups))
	for i, gs := range groups {
		total = total + len(gs)
		whitelist[i] = make([]uint32, len(gs))
		for j, g := range gs {
			if g == nil || len(g.Txs) == 0 {
				continue
			}
			totalTransitions = totalTransitions + len(g.Txs)
			whitelist[i][j] = g.Txs[0]
		}
	}
	tims[0] = time.Since(begintime)

	t0 := time.Now()

	txs, g, flags, tetails, totals := wrapper.Detect(arb.arbitrator, whitelist)
	tims[1] = time.Since(t0)

	begintime = time.Now()

	l, r := wrapper.ExportTxs(arb.arbitrator)
	tims[2] = time.Since(begintime)
	return txs, g, flags, l, r, tims, total, tetails, totals
}
