package types

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/concurrenturl/interfaces"

	// eucommon "github.com/arcology-network/eu/common"
	"github.com/arcology-network/eu/execution"
	eushared "github.com/arcology-network/eu/shared"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ExecMessagers struct {
	Snapshot *interfaces.Datastore
	Config   *execution.Config
	Sequence *types.ExecutingSequence
	// Uuid     uint64
	// SerialID int
	// Total    int
	Debug bool
}

type ExecutionResponse struct {
	AccessRecords   []*eushared.TxAccessRecords
	EuResults       []*eushared.EuResult
	Receipts        []*evmTypes.Receipt
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
	ExecutingLogs   []string
}
