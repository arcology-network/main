package statesync

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
)

type P2pClient interface {
	ID() string
	Broadcast(msg *actor.Message)
	Request(peer string, msg *actor.Message)
	Response(peer string, msg *actor.Message)
	OnConnClosed(cb func(id string))
}

type StorageRpc interface {
	GetSyncStatus() *mtypes.SyncStatus
	GetSyncPoint(height uint64) *mtypes.SyncPoint

	WriteSlice(slice *mtypes.SyncDataResponse)
	ReadSlice(request *mtypes.SyncDataRequest) *mtypes.SyncDataResponse
	ApplyData(request *mtypes.SyncDataRequest)
	RewriteMeta()
}
