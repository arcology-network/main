package types

import (
	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/concurrenturl/interfaces"
	evmCommon "github.com/arcology-network/evm/common"
	evmTypes "github.com/arcology-network/evm/core/types"
	adaptor "github.com/arcology-network/vm-adaptor/execution"
)

type ExecMessagers struct {
	Snapshot *interfaces.Datastore
	Config   *adaptor.Config
	Sequence *types.ExecutingSequence
	// Uuid     uint64
	// SerialID int
	// Total    int
	Debug bool
}

type ExecutionResponse struct {
	AccessRecords   []*types.TxAccessRecords
	EuResults       []*types.EuResult
	Receipts        []*evmTypes.Receipt
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
	ExecutingLogs   []string
}
