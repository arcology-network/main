package scheduler

import (
	"math/big"
	"strings"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	schdv1 "github.com/arcology-network/main/modules/scheduler"
)

type processContext struct {
	executor   *ExecClient
	arbitrator schdv1.Arbitrator

	// Per block data.
	timestamp           *big.Int
	txHash2Callee       map[ethcmn.Hash]ethcmn.Address
	txHash2IdBiMap      *cmncmn.BiMap[ethcmn.Hash, uint32]
	txHash2Gas          map[ethcmn.Hash]uint64
	executedLastGen     []*ethcmn.Hash
	executedHashLastGen ethcmn.Hash
	executed            []*ethcmn.Hash
	executedHash        ethcmn.Hash
	deletedDict         map[ethcmn.Hash]struct{}
	spawnedRelations    []*cmntyp.SpawnedRelation
	txId                uint32

	// Parameters for executor.
	msgTemplate *actor.Message
	logger      *actor.WorkerThreadLogger
	parallelism int
	generation  int
	batch       int

	// Results collected for scheduler.
	newContracts []ethcmn.Address
	conflicts    map[ethcmn.Address][]ethcmn.Address
}

func createProcessContext() *processContext {
	return &processContext{
		txHash2Callee:       make(map[ethcmn.Hash]ethcmn.Address),
		txHash2IdBiMap:      cmncmn.NewBiMap[ethcmn.Hash, uint32](),
		txHash2Gas:          make(map[ethcmn.Hash]uint64),
		executedLastGen:     make([]*ethcmn.Hash, 0, 50000),
		executedHashLastGen: ethcmn.Hash{},
		executed:            make([]*ethcmn.Hash, 0, 50000),
		executedHash:        ethcmn.Hash{},
		deletedDict:         make(map[ethcmn.Hash]struct{}),
		conflicts:           make(map[ethcmn.Address][]ethcmn.Address),
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
	c.arbitrator = schdv1.NewRpcClientArbitrate()
}

func (c *processContext) onNewBlock() {
	c.txHash2Callee = make(map[ethcmn.Hash]ethcmn.Address)
	c.txHash2IdBiMap = cmncmn.NewBiMap[ethcmn.Hash, uint32]()
	c.txHash2Gas = make(map[ethcmn.Hash]uint64)
	c.executedLastGen = c.executedLastGen[:0]
	c.executedHashLastGen = ethcmn.Hash{}
	c.executed = c.executed[:0]
	c.executedHash = ethcmn.Hash{}
	c.deletedDict = make(map[ethcmn.Hash]struct{})
	c.spawnedRelations = c.spawnedRelations[:0]
	c.txId = 1
	c.generation = 0
	c.batch = 0
	c.newContracts = c.newContracts[:0]
	c.conflicts = make(map[ethcmn.Address][]ethcmn.Address)
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
