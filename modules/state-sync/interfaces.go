package statesync

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
)

type P2pClient interface {
	ID() string
	Broadcast(msg *actor.Message)
	Request(peer string, msg *actor.Message)
	Response(peer string, msg *actor.Message)
	OnConnClosed(cb func(id string))
}

type StorageRpc interface {
	GetSyncStatus() *cmntyp.SyncStatus
	GetSyncPoint(height uint64) *cmntyp.SyncPoint

	WriteSlice(slice *cmntyp.SyncDataResponse)
	ReadSlice(request *cmntyp.SyncDataRequest) *cmntyp.SyncDataResponse
	ApplyData(request *cmntyp.SyncDataRequest)
	RewriteMeta()
}
