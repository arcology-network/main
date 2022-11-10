package statesync

import (
	cmntyp "github.com/arcology-network/common-lib/types"
	intf "github.com/arcology-network/component-lib/interface"
)

type DefaultStorageRpc struct{}

func NewDefaultStorageRpc() *DefaultStorageRpc {
	return &DefaultStorageRpc{}
}

func (rpc *DefaultStorageRpc) GetSyncStatus() *cmntyp.SyncStatus {
	var na int
	var status cmntyp.SyncStatus
	intf.Router.Call("statesyncstore", "GetSyncStatus", &na, &status)
	return &status
}

func (rpc *DefaultStorageRpc) GetSyncPoint(height uint64) *cmntyp.SyncPoint {
	var sp cmntyp.SyncPoint
	intf.Router.Call("statesyncstore", "GetSyncPoint", &height, &sp)
	return &sp
}

func (rpc *DefaultStorageRpc) WriteSlice(slice *cmntyp.SyncDataResponse) {
	var na int
	intf.Router.Call("statesyncstore", "WriteSlice", slice, &na)
}

func (rpc *DefaultStorageRpc) ReadSlice(request *cmntyp.SyncDataRequest) *cmntyp.SyncDataResponse {
	var response cmntyp.SyncDataResponse
	intf.Router.Call("statesyncstore", "ReadSlice", request, &response)
	return &response
}

func (rpc *DefaultStorageRpc) ApplyData(request *cmntyp.SyncDataRequest) {
	var na int
	intf.Router.Call("urlstore", "ApplyData", request, &na)
}

func (rpc *DefaultStorageRpc) RewriteMeta() {
	var na int
	intf.Router.Call("urlstore", "RewriteMeta", &na, &na)
}
