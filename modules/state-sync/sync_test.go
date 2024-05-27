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

package statesync

import (
	"fmt"
	"testing"
	"time"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	brokerpk "github.com/arcology-network/streamer/broker"
)

type syncer struct{}

func (s *syncer) Consume(data interface{}) {
	fmt.Printf("state sync done: %v\n", data)
}

func TestBasic(t *testing.T) {
	// 1 to 3 sync.
	broker := brokerpk.NewStatefulStreamer()
	broker.RegisterProducer(brokerpk.NewDefaultProducer(
		"syncClient",
		[]string{actor.MsgStateSyncDone},
		[]int{1},
	))
	broker.RegisterConsumer(brokerpk.NewDefaultConsumer(
		"syncer",
		[]string{actor.MsgStateSyncDone},
		brokerpk.NewDisjunctions(&syncer{}, 1),
	))
	broker.Serve()

	syncClient := NewSyncClient(1, "s1").(*SyncClient)
	syncClient.Init("client", broker)

	syncServers := []*SyncServer{
		NewSyncServer(1, "s2").(*SyncServer),
		NewSyncServer(1, "s3").(*SyncServer),
		NewSyncServer(1, "s4").(*SyncServer),
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

	storageSrvs := []*storageMock{
		newStorageMock(&mtypes.SyncStatus{
			SyncPoint: 0,
			Height:    0,
		}),
		newStorageMock(&mtypes.SyncStatus{
			SyncPoint: 100,
			Height:    110,
		}),
		newStorageMock(&mtypes.SyncStatus{
			SyncPoint: 100,
			Height:    120,
		}),
		newStorageMock(&mtypes.SyncStatus{
			SyncPoint: 100,
			Height:    130,
		}),
	}

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
			Data: uint64(125),
		},
	})

	time.Sleep(30 * time.Second)
}
