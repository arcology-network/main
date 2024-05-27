/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package statesync

import (
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type storageMock struct {
	status *mtypes.SyncStatus
}

func newStorageMock(status *mtypes.SyncStatus) *storageMock {
	return &storageMock{
		status: status,
	}
}

func (m *storageMock) GetSyncStatus() *mtypes.SyncStatus {
	return m.status
}

func (m *storageMock) GetSyncPoint(height uint64) *mtypes.SyncPoint {
	if m.status == nil || m.status.SyncPoint == 0 {
		return nil
	}

	return &mtypes.SyncPoint{
		From:   0,
		To:     m.status.SyncPoint,
		Slices: make([]evmCommon.Hash, mtypes.SlicePerSyncPoint),
		Parent: &mtypes.ParentInfo{},
	}
}

func (m *storageMock) WriteSlice(response *mtypes.SyncDataResponse) {
	fmt.Printf("StorageMock: write slice [%v]\n", response)
}

func (m *storageMock) ReadSlice(request *mtypes.SyncDataRequest) *mtypes.SyncDataResponse {
	if m.status == nil || m.status.Height < request.To {
		return nil
	}

	return &mtypes.SyncDataResponse{
		SyncDataRequest: *request,
		Data:            make([]byte, request.Slice+1),
	}
}

func (m *storageMock) ApplyData(request *mtypes.SyncDataRequest) {
	fmt.Printf("StorageMock: apply data [%v]\n", request)
}

func (m *storageMock) RewriteMeta() {}
