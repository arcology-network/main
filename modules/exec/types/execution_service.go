package types

import (
	"github.com/arcology-network/storage-committer/interfaces"

	eushared "github.com/arcology-network/eu/shared"
	evmAdaptorCommon "github.com/arcology-network/evm-adaptor/common"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type ExecMessagers struct {
	Snapshot *interfaces.Datastore
	Config   *evmAdaptorCommon.Config
	Sequence *mtypes.ExecutingSequence
	Debug    bool
}

type ExecutionResponse struct {
	AccessRecords   []*eushared.TxAccessRecords
	EuResults       []*eushared.EuResult
	Receipts        []*evmTypes.Receipt
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
	ExecutingLogs   []string
}
