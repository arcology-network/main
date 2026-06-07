package types

import (
	// statestore "github.com/arcology-network/state-engine"
	stateengine "github.com/arcology-network/state-engine/state/cache"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type Initialization struct {
	Store             *stateengine.ExecutionStateStore
	BlockStart        *actor.BlockStart
	ChainConfig       *params.ChainConfig
	ParentInformation *ParentInfo
}

type SchdState struct {
	Height            uint64
	NewContracts      []evmCommon.Address
	ConflictionLefts  []evmCommon.Address
	ConflictionRights []evmCommon.Address

	ConflictionLeftSigns  [][4]byte
	ConflictionRightSigns [][4]byte
}
