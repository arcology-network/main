package exec

import (
	eupk "github.com/arcology-network/eu"

	eucommon "github.com/arcology-network/eu/common"
	exetyp "github.com/arcology-network/main/modules/exec/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/storage-committer/interfaces"
	stgcommitter "github.com/arcology-network/storage-committer/storage/committer"
	"github.com/arcology-network/streamer/actor"
)

type ExecutionImpl interface {
	Init(eu *eupk.EU, url *stgcommitter.StateCommitter)
	SetDB(db *interfaces.ReadOnlyStore)
	Exec(sequence *mtypes.ExecutingSequence, config *eucommon.Config, logger *actor.WorkerThreadLogger, gatherExeclog bool) (*exetyp.ExecutionResponse, error)
}
