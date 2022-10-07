package statesync

import (
	"fmt"

	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
)

type storageMock struct {
	status *cmntyp.SyncStatus
}

func newStorageMock(status *cmntyp.SyncStatus) *storageMock {
	return &storageMock{
		status: status,
	}
}

func (m *storageMock) GetSyncStatus() *cmntyp.SyncStatus {
	return m.status
}

func (m *storageMock) GetSyncPoint(height uint64) *cmntyp.SyncPoint {
	if m.status == nil || m.status.SyncPoint == 0 {
		return nil
	}

	return &cmntyp.SyncPoint{
		From:   0,
		To:     m.status.SyncPoint,
		Slices: make([]ethcmn.Hash, cmntyp.SlicePerSyncPoint),
	}
}

func (m *storageMock) WriteSlice(response *cmntyp.SyncDataResponse) {
	fmt.Printf("StorageMock: write slice [%v]\n", response)
}

func (m *storageMock) ReadSlice(request *cmntyp.SyncDataRequest) *cmntyp.SyncDataResponse {
	if m.status == nil || m.status.Height < request.To {
		return nil
	}

	return &cmntyp.SyncDataResponse{
		SyncDataRequest: *request,
		Data:            make([]byte, request.Slice+1),
	}
}

func (m *storageMock) ApplyData(request *cmntyp.SyncDataRequest) {
	fmt.Printf("StorageMock: apply data [%v]\n", request)
}

func (m *storageMock) RewriteMeta() {}
