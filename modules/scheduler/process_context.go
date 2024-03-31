package scheduler

import (
	"math/big"
	"strings"

	cmncmn "github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	evmCommon "github.com/ethereum/go-ethereum/common"

	// schdv1 "github.com/arcology-network/main/modules/scheduler"
	mtypes "github.com/arcology-network/main/types"
)

type processContext struct {
	executor   *ExecClient
	arbitrator Arbitrator

	// Per block data.
	timestamp     *big.Int
	txHash2Callee map[evmCommon.Hash]evmCommon.Address
	txHash2Sign   map[evmCommon.Hash][4]byte

	txHash2IdBiMap *cmncmn.BiMap[evmCommon.Hash, uint32]
	txHash2Gas     map[evmCommon.Hash]uint64

	executed    []evmCommon.Hash
	deletedDict map[evmCommon.Hash]struct{}

	txId uint32

	// Parameters for executor.
	msgTemplate *actor.Message
	logger      *actor.WorkerThreadLogger
	parallelism int
	generation  int
	// batch       int

	height uint64

	// Results collected for scheduler.
	newContracts []evmCommon.Address
	conflicts    *mtypes.ConflictInfos
}

func createProcessContext() *processContext {
	return &processContext{
		txHash2Callee:  make(map[evmCommon.Hash]evmCommon.Address),
		txHash2Sign:    make(map[evmCommon.Hash][4]byte),
		txHash2IdBiMap: cmncmn.NewBiMap[evmCommon.Hash, uint32](),
		txHash2Gas:     make(map[evmCommon.Hash]uint64),

		deletedDict: make(map[evmCommon.Hash]struct{}),
		conflicts:   mtypes.NewConflictInfos(), //make(map[evmCommon.Address][]evmCommon.Address),
		txId:        1,
	}
}

func (c *processContext) init(execBatchSize int) {
	var execSvcs []string
	for _, svc := range intf.Router.GetAvailableServices() {
		if strings.HasPrefix(svc, "executor") {
			execSvcs = append(execSvcs, svc)
		}
	}
	c.executor = NewExecClient(execSvcs, execBatchSize)
	c.arbitrator = NewRpcClientArbitrate()
}

func (c *processContext) onNewBlock(height uint64) {
	c.txHash2Callee = make(map[evmCommon.Hash]evmCommon.Address)
	c.txHash2Sign = make(map[evmCommon.Hash][4]byte)
	c.txHash2IdBiMap = cmncmn.NewBiMap[evmCommon.Hash, uint32]()
	c.txHash2Gas = make(map[evmCommon.Hash]uint64)
	c.executed = c.executed[:0]
	c.deletedDict = make(map[evmCommon.Hash]struct{})
	c.txId = 1
	c.generation = 0
	c.newContracts = c.newContracts[:0]
	c.conflicts.Reset()

	c.height = height
}

func (c *processContext) onNewGeneration() {
	c.generation++
}
