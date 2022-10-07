package statesync

import (
	"fmt"
	"testing"
	"time"

	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/streamer"
)

type syncer struct{}

func (s *syncer) Consume(data interface{}) {
	fmt.Printf("state sync done: %v\n", data)
}

func TestBasic(t *testing.T) {
	// 1 to 3 sync.
	broker := streamer.NewStatefulStreamer()
	broker.RegisterProducer(streamer.NewDefaultProducer(
		"syncClient",
		[]string{actor.MsgStateSyncDone},
		[]int{1},
	))
	broker.RegisterConsumer(streamer.NewDefaultConsumer(
		"syncer",
		[]string{actor.MsgStateSyncDone},
		streamer.NewDisjunctions(&syncer{}, 1),
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
		newStorageMock(&cmntyp.SyncStatus{
			SyncPoint: 0,
			Height:    0,
		}),
		newStorageMock(&cmntyp.SyncStatus{
			SyncPoint: 100,
			Height:    110,
		}),
		newStorageMock(&cmntyp.SyncStatus{
			SyncPoint: 100,
			Height:    120,
		}),
		newStorageMock(&cmntyp.SyncStatus{
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
