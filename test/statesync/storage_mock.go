package statesync

import (
	"context"
	"fmt"

	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/main/modules/storage"
)

type storageMockV2 struct {
	ssStore *storage.StateSyncStore
}

func newStorageMockV2(ssStore *storage.StateSyncStore) *storageMockV2 {
	return &storageMockV2{
		ssStore: ssStore,
	}
}

func (mock *storageMockV2) GetSyncStatus() *cmntyp.SyncStatus {
	var na int
	var status cmntyp.SyncStatus
	mock.ssStore.GetSyncStatus(context.Background(), &na, &status)
	return &status
}

func (mock *storageMockV2) GetSyncPoint(height uint64) *cmntyp.SyncPoint {
	var sp cmntyp.SyncPoint
	mock.ssStore.GetSyncPoint(context.Background(), &height, &sp)
	return &sp
}

func (mock *storageMockV2) WriteSlice(response *cmntyp.SyncDataResponse) {
	var na int
	mock.ssStore.WriteSlice(context.Background(), response, &na)
}

func (mock *storageMockV2) ReadSlice(request *cmntyp.SyncDataRequest) *cmntyp.SyncDataResponse {
	var response cmntyp.SyncDataResponse
	mock.ssStore.ReadSlice(context.Background(), request, &response)
	return &response
}

func (mock *storageMockV2) ApplyData(request *cmntyp.SyncDataRequest) {
	status := mock.GetSyncStatus()
	fmt.Printf("storageMockV2.ApplyData, request: %v\n", request)
	if request.To-request.From > 1 {
		var sp cmntyp.SyncPoint
		mock.ssStore.InitSyncPoint(context.Background(), &request.To, &sp)
		status.SyncPoint = request.To
	}
	status.Height = request.To
	var na int
	mock.ssStore.SetSyncStatus(context.Background(), status, &na)
}

func (mock *storageMockV2) RewriteMeta() {}
