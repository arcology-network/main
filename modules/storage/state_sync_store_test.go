//go:build !CI

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

package storage

import (
	"context"
	"fmt"
	"testing"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
)

func TestSliceKey(t *testing.T) {
	store := NewStateSyncStore(1, "statesyncstore").(*StateSyncStore)
	key := store.sliceKey(&mtypes.SyncDataRequest{
		From:  0,
		To:    1024,
		Slice: 65535,
	})
	t.Log(key)
}

func TestSetGetSyncStatus(t *testing.T) {
	worker := NewStateSyncStore(1, "statesyncstore")
	worker.(actor.Configurable).Config(map[string]interface{}{
		"slice_db_root":       "./slices/",
		"sync_point_root":     "./sync_point/",
		"sync_point_interval": float64(65536),
	})
	store := worker.(*StateSyncStore)

	status := &mtypes.SyncStatus{
		Id:        "node1",
		SyncPoint: 1024,
		Height:    1080,
	}
	var na int
	store.SetSyncStatus(context.Background(), status, &na)

	var s mtypes.SyncStatus
	store.GetSyncStatus(context.Background(), &na, &s)
	t.Log(s)
}

func TestSetGetSyncPoint(t *testing.T) {
	worker := NewStateSyncStore(1, "statesyncstore")
	worker.(actor.Configurable).Config(map[string]interface{}{
		"slice_db_root":       "./slices1/",
		"sync_point_root":     "./sync_point1/",
		"sync_point_interval": float64(65536),
	})
	store := worker.(*StateSyncStore)

	sp := &mtypes.SyncPoint{
		From: 0,
		To:   1024,
	}
	var na int
	store.SetSyncPoint(context.Background(), sp, &na)

	var s mtypes.SyncPoint
	store.GetSyncPoint(context.Background(), &sp.To, &s)
	t.Log(s)
}

func TestFormatQueryCriteria(t *testing.T) {
	t.Log(fmt.Sprintf("%s%02x", RootPrefix, []byte{byte(1)}))
}
