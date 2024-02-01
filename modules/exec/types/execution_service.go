package types

import (
	"github.com/arcology-network/concurrenturl/interfaces"

	"github.com/arcology-network/eu/execution"
	eushared "github.com/arcology-network/eu/shared"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ExecMessagers struct {
	Snapshot *interfaces.Datastore
	Config   *execution.Config
	Sequence *mtypes.ExecutingSequence
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
