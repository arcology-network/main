package scheduler

import (
	"math/big"
	"strings"

	cmncmn "github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	evmCommon "github.com/ethereum/go-ethereum/common"
	// schdv1 "github.com/arcology-network/main/modules/scheduler"
)

type processContext struct {
	executor   *ExecClient
	arbitrator Arbitrator

	// Per block data.
	timestamp           *big.Int
	txHash2Callee       map[evmCommon.Hash]evmCommon.Address
	txHash2IdBiMap      *cmncmn.BiMap[evmCommon.Hash, uint32]
	txHash2Gas          map[evmCommon.Hash]uint64
	executedLastGen     []*evmCommon.Hash
	executedHashLastGen evmCommon.Hash
	executed            []*evmCommon.Hash
	executedHash        evmCommon.Hash
	deletedDict         map[evmCommon.Hash]struct{}
	// spawnedRelations    []*cmntyp.SpawnedRelation
	txId uint32

	// Parameters for executor.
	msgTemplate *actor.Message
	logger      *actor.WorkerThreadLogger
	parallelism int
	generation  int
	batch       int

	// Results collected for scheduler.
	newContracts []evmCommon.Address
	conflicts    map[evmCommon.Address][]evmCommon.Address
}

func createProcessContext() *processContext {
	return &processContext{
		txHash2Callee:       make(map[evmCommon.Hash]evmCommon.Address),
		txHash2IdBiMap:      cmncmn.NewBiMap[evmCommon.Hash, uint32](),
		txHash2Gas:          make(map[evmCommon.Hash]uint64),
		executedLastGen:     make([]*evmCommon.Hash, 0, 50000),
		executedHashLastGen: evmCommon.Hash{},
		executed:            make([]*evmCommon.Hash, 0, 50000),
		executedHash:        evmCommon.Hash{},
		deletedDict:         make(map[evmCommon.Hash]struct{}),
		conflicts:           make(map[evmCommon.Address][]evmCommon.Address),
		txId:                1,
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

func (c *processContext) onNewBlock() {
	c.txHash2Callee = make(map[evmCommon.Hash]evmCommon.Address)
	c.txHash2IdBiMap = cmncmn.NewBiMap[evmCommon.Hash, uint32]()
	c.txHash2Gas = make(map[evmCommon.Hash]uint64)
	c.executedLastGen = c.executedLastGen[:0]
	c.executedHashLastGen = evmCommon.Hash{}
	c.executed = c.executed[:0]
	c.executedHash = evmCommon.Hash{}
	c.deletedDict = make(map[evmCommon.Hash]struct{})
	// c.spawnedRelations = c.spawnedRelations[:0]
	c.txId = 1
	c.generation = 0
	c.batch = 0
	c.newContracts = c.newContracts[:0]
	c.conflicts = make(map[evmCommon.Address][]evmCommon.Address)
}

func (c *processContext) onNewGeneration() {
	c.generation++
	c.batch = 0
	c.executedLastGen = c.executed
	c.executedHashLastGen = c.executedHash
}

func (c *processContext) onNewBatch() {
	c.batch++
}
