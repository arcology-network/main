package types

import (
	statestore "github.com/arcology-network/storage-committer"
	"github.com/arcology-network/streamer/actor"
	"github.com/ethereum/go-ethereum/params"
)

type Initialization struct {
	Store             *statestore.StateStore
	BlockStart        *actor.BlockStart
	ChainConfig       *params.ChainConfig
	ParentInformation *ParentInfo
}
