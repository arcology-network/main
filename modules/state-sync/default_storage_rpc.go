package statesync

import (
	mtypes "github.com/arcology-network/main/types"
	intf "github.com/arcology-network/streamer/interface"
)

type DefaultStorageRpc struct{}

func NewDefaultStorageRpc() *DefaultStorageRpc {
	return &DefaultStorageRpc{}
}

func (rpc *DefaultStorageRpc) GetSyncStatus() *mtypes.SyncStatus {
	var na int
	var status mtypes.SyncStatus
	intf.Router.Call("statesyncstore", "GetSyncStatus", &na, &status)
	return &status
}

func (rpc *DefaultStorageRpc) GetSyncPoint(height uint64) *mtypes.SyncPoint {
	var sp mtypes.SyncPoint
	intf.Router.Call("statesyncstore", "GetSyncPoint", &height, &sp)
	return &sp
}

func (rpc *DefaultStorageRpc) WriteSlice(slice *mtypes.SyncDataResponse) {
	var na int
	intf.Router.Call("statesyncstore", "WriteSlice", slice, &na)
}

func (rpc *DefaultStorageRpc) ReadSlice(request *mtypes.SyncDataRequest) *mtypes.SyncDataResponse {
	var response mtypes.SyncDataResponse
	intf.Router.Call("statesyncstore", "ReadSlice", request, &response)
	return &response
}

func (rpc *DefaultStorageRpc) ApplyData(request *mtypes.SyncDataRequest) {
	var na int
	intf.Router.Call("urlstore", "ApplyData", request, &na)
}

func (rpc *DefaultStorageRpc) RewriteMeta() {
	var na int
	intf.Router.Call("urlstore", "RewriteMeta", &na, &na)
}
