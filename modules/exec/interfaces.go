package exec

import (
	eupk "github.com/arcology-network/eu"

	eucommon "github.com/arcology-network/eu/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	mtypes "github.com/arcology-network/main/types"
	ccurl "github.com/arcology-network/storage-committer"
	"github.com/arcology-network/storage-committer/interfaces"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type SnapshotDict interface {
	Reset(snapshot *interfaces.Datastore)
	AddItem(precedingHash evmCommon.Hash, size int, snapshot *interfaces.Datastore)
	Query(precedings []*evmCommon.Hash) (*interfaces.Datastore, []*evmCommon.Hash)
}

type ExecutionImpl interface {
	Init(eu *eupk.EU, url *ccurl.StateCommitter)
	SetDB(db *interfaces.Datastore)
	Exec(sequence *mtypes.ExecutingSequence, config *eucommon.Config, logger *actor.WorkerThreadLogger, gatherExeclog bool) (*exetyp.ExecutionResponse, error)
}
