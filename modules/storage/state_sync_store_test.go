//go:build !CI

package storage

import (
	"context"
	"fmt"
	"testing"

	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/streamer/actor"
)

func TestSliceKey(t *testing.T) {
	store := NewStateSyncStore(1, "statesyncstore").(*StateSyncStore)
	key := store.sliceKey(&cmntyp.SyncDataRequest{
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

	status := &cmntyp.SyncStatus{
		Id:        "node1",
		SyncPoint: 1024,
		Height:    1080,
	}
	var na int
	store.SetSyncStatus(context.Background(), status, &na)

	var s cmntyp.SyncStatus
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

	sp := &cmntyp.SyncPoint{
		From: 0,
		To:   1024,
	}
	var na int
	store.SetSyncPoint(context.Background(), sp, &na)

	var s cmntyp.SyncPoint
	store.GetSyncPoint(context.Background(), &sp.To, &s)
	t.Log(s)
}

func TestFormatQueryCriteria(t *testing.T) {
	t.Log(fmt.Sprintf("%s%02x", RootPrefix, []byte{byte(1)}))
}
