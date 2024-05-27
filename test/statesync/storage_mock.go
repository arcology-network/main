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
	"context"
	"fmt"

	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
)

type storageMockV2 struct {
	ssStore *storage.StateSyncStore
}

func newStorageMockV2(ssStore *storage.StateSyncStore) *storageMockV2 {
	return &storageMockV2{
		ssStore: ssStore,
	}
}

func (mock *storageMockV2) GetSyncStatus() *mtypes.SyncStatus {
	var na int
	var status mtypes.SyncStatus
	mock.ssStore.GetSyncStatus(context.Background(), &na, &status)
	return &status
}

func (mock *storageMockV2) GetSyncPoint(height uint64) *mtypes.SyncPoint {
	var sp mtypes.SyncPoint
	mock.ssStore.GetSyncPoint(context.Background(), &height, &sp)
	return &sp
}

func (mock *storageMockV2) WriteSlice(response *mtypes.SyncDataResponse) {
	var na int
	mock.ssStore.WriteSlice(context.Background(), response, &na)
}

func (mock *storageMockV2) ReadSlice(request *mtypes.SyncDataRequest) *mtypes.SyncDataResponse {
	var response mtypes.SyncDataResponse
	mock.ssStore.ReadSlice(context.Background(), request, &response)
	return &response
}

func (mock *storageMockV2) ApplyData(request *mtypes.SyncDataRequest) {
	status := mock.GetSyncStatus()
	fmt.Printf("storageMockV2.ApplyData, request: %v\n", request)
	if request.To-request.From > 1 {
		var sp mtypes.SyncPoint
		mock.ssStore.InitSyncPoint(context.Background(), &request.To, &sp)
		status.SyncPoint = request.To
	}
	status.Height = request.To
	var na int
	mock.ssStore.SetSyncStatus(context.Background(), status, &na)
}

func (mock *storageMockV2) RewriteMeta() {}
