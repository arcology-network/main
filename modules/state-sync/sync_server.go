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

	"github.com/arcology-network/main/modules/p2p"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
)

type SyncServer struct {
	actor.WorkerThread

	p2pClient  P2pClient
	storageRpc StorageRpc
}

func NewSyncServer(concurrency int, groupId string) actor.IWorkerEx {
	srv := &SyncServer{
		p2pClient:  p2p.NewP2pClient(concurrency, "p2p.client").(P2pClient),
		storageRpc: NewDefaultStorageRpc(),
	}
	srv.Set(concurrency, groupId)
	return srv
}

func (srv *SyncServer) TestOnlySetP2pClient(p2pClient P2pClient) {
	srv.p2pClient = p2pClient
}

func (srv *SyncServer) TestOnlySetStorageRpc(storageRpc StorageRpc) {
	srv.storageRpc = storageRpc
}

func (srv *SyncServer) Inputs() ([]string, bool) {
	return []string{
		actor.MsgP2pRequest,
	}, false
}

func (srv *SyncServer) Outputs() map[string]int {
	return map[string]int{}
}

func (srv *SyncServer) OnStart() {}

func (srv *SyncServer) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgP2pRequest:
		p2pMessage := msg.Data.(*p2p.P2pMessage)
		msg = p2pMessage.Message
		switch msg.Name {
		case actor.MsgSyncStatusRequest:
			status := srv.storageRpc.GetSyncStatus()
			status.Id = srv.p2pClient.ID()
			fmt.Printf("SyncServer[%s]: handle sync status request [%v]\n", srv.Groupid, status)
			srv.p2pClient.Response(p2pMessage.Sender, &actor.Message{
				Name: actor.MsgSyncStatusResponse,
				Data: status,
			})
		case actor.MsgSyncPointRequest:
			sp := srv.storageRpc.GetSyncPoint(msg.Data.(uint64))
			fmt.Printf("SyncServer[%s]: handle sync point request [%v]\n", srv.Groupid, sp)
			srv.p2pClient.Response(p2pMessage.Sender, &actor.Message{
				Name: actor.MsgSyncPointResponse,
				Data: sp,
			})
		case actor.MsgSyncDataRequest:
			data := srv.storageRpc.ReadSlice(msg.Data.(*mtypes.SyncDataRequest))
			fmt.Printf("SyncServer[%s]: handle sync data request [%v]\n", srv.Groupid, msg.Data.(*mtypes.SyncDataRequest))
			srv.p2pClient.Response(p2pMessage.Sender, &actor.Message{
				Name: actor.MsgSyncDataResponse,
				Data: data,
			})
		}
	}
	return nil
}
