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
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	cmnst "github.com/arcology-network/main/components/storage"
	statesync "github.com/arcology-network/main/modules/state-sync"
	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	brokerpk "github.com/arcology-network/streamer/broker"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

var (
	latestHeight    = uint64(41)
	latestSyncPoint = uint64(32)
)

func TestMakeSyncPoint(t *testing.T) {
	var keys []string
	var values [][]byte
	for i := 0; i < int(latestHeight); i++ {
		keys = append(keys, randomHexString(4))
		values = append(values, []byte(randomHexString(4)))
	}

	ssStore := makeSyncPoint(t, "./data/statesyncstore", keys, values)
	var na int
	var status mtypes.SyncStatus
	ssStore.GetSyncStatus(context.Background(), &na, &status)
	t.Log(status)

	var sp mtypes.SyncPoint
	height := latestSyncPoint
	ssStore.GetSyncPoint(context.Background(), &height, &sp)
	t.Log(sp)
}

type syncer struct {
	keys   []string
	values [][]byte
	mock   *storageMockV2
}

func (s *syncer) Consume(data interface{}) {
	fmt.Printf("state sync done: %v\n", data)
	sp := s.mock.ssStore.TestOnlyGetSyncPointDB()
	for i := 0; i < int(latestSyncPoint); i++ {
		v, err := sp.Get(s.keys[i])
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(v, s.values[i]) {
			panic(fmt.Sprintf("v = %v, want %v", v, s.values[i]))
		}
	}
	for i := int(latestSyncPoint); i < int(latestHeight); i++ {
		v, err := sp.Get(s.keys[i])
		if err == nil && len(v) != 0 {
			panic("value should not be found")
		}
	}
}

func (s *syncer) init(keys []string, values [][]byte, mock *storageMockV2) {
	s.keys = keys
	s.values = values
	s.mock = mock
}

func TestStateSync(t *testing.T) {
	var keys []string
	var values [][]byte
	for i := 0; i < int(latestHeight); i++ {
		keys = append(keys, randomHexString(4))
		values = append(values, []byte(randomHexString(4)))
	}

	broker := brokerpk.NewStatefulStreamer()
	syncer := &syncer{}
	broker.RegisterProducer(brokerpk.NewDefaultProducer(
		"syncClient",
		[]string{actor.MsgStateSyncDone},
		[]int{1},
	))
	broker.RegisterConsumer(brokerpk.NewDefaultConsumer(
		"syncer",
		[]string{actor.MsgStateSyncDone},
		brokerpk.NewDisjunctions(syncer, 1),
	))
	broker.Serve()

	syncClient := statesync.NewSyncClient(1, "s1").(*statesync.SyncClient)
	syncClient.Init("client", broker)

	syncServers := []*statesync.SyncServer{
		statesync.NewSyncServer(1, "s2").(*statesync.SyncServer),
		statesync.NewSyncServer(1, "s3").(*statesync.SyncServer),
		statesync.NewSyncServer(1, "s4").(*statesync.SyncServer),
	}

	p2pClients := []*p2pClientMock{
		newP2pClientMock("c1", syncClient),
		newP2pClientMock("c2", syncServers[0]),
		newP2pClientMock("c3", syncServers[1]),
		newP2pClientMock("c4", syncServers[2]),
	}
	sw := newSwitchMock()
	for _, client := range p2pClients {
		sw.add(client)
	}

	storageSrvs := []*storageMockV2{
		newStorageMockV2(makeSyncPoint(t, "./data/client", nil, nil)),
		newStorageMockV2(makeSyncPoint(t, "./data/server1", keys, values)),
		newStorageMockV2(makeSyncPoint(t, "./data/server2", keys, values)),
		newStorageMockV2(makeSyncPoint(t, "./data/server3", keys, values)),
	}
	syncer.init(keys, values, storageSrvs[0])

	syncClient.TestOnlySetP2pClient(p2pClients[0])
	syncClient.TestOnlySetStorageRpc(storageSrvs[0])
	for i, srv := range syncServers {
		srv.TestOnlySetP2pClient(p2pClients[i+1])
		srv.TestOnlySetStorageRpc(storageSrvs[i+1])
	}

	syncClient.OnStart()
	syncClient.OnMessageArrived([]*actor.Message{
		{
			Name: actor.MsgStateSyncStart,
			Data: uint64(latestHeight),
		},
	})

	time.Sleep(30 * time.Second)
}

func makeSyncPoint(t *testing.T, root string, keys []string, values [][]byte) *storage.StateSyncStore {
	os.RemoveAll(root + "/syncpoint/data/")
	os.RemoveAll(root + "/slices/")
	ssStore := storage.TestOnlyNewStateSyncStore(1, "statesyncstore").(*storage.StateSyncStore)
	ssStore.Config(map[string]interface{}{
		"slice_db_root":       root + "/slices/",
		"sync_point_root":     root + "/syncpoint/data/",
		"sync_point_interval": float64(latestSyncPoint),
	})

	for i := range keys {
		ssStore.OnMessageArrived([]*actor.Message{
			{
				Name:   actor.CombinedName(actor.MsgUrlUpdate, actor.MsgAcctHash),
				Height: uint64(i + 1),
				Data: &actor.CombinerElements{
					Msgs: map[string]*actor.Message{
						actor.MsgUrlUpdate: {
							Data: &cmnst.UrlUpdate{
								Keys: []string{
									keys[i],
								},
								EncodedValues: [][]byte{
									values[i],
								},
							},
						},
						actor.MsgAcctHash: {
							Data: evmCommon.BytesToHash([]byte(fmt.Sprintf("hash%d", i))),
						},
					},
				},
			},
		})
	}
	return ssStore
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomHexString(nbytes int) string {
	b := make([]byte, nbytes)
	rnd.Read(b)
	return fmt.Sprintf("%x", b)
}
