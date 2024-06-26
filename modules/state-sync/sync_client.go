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
	"sync"
	"time"

	"github.com/arcology-network/main/modules/p2p"
	"github.com/arcology-network/main/modules/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

/*
MessageType 			DataType			Meaning
MsgStateSyncStart		uint64				target height
MsgSyncStatusRequest	nil
MsgSyncStatusResponse	SyncStatus
MsgSyncPointRequest		uint64				sync point height
MsgSyncPointResponse	SyncPoint
MsgSyncDataRequest		SyncDataRequest		from/to/slice
MsgSyncDataResponse		SyncDataResponse
*/

type SyncClient struct {
	actor.WorkerThread

	peers map[string]*mtypes.SyncStatus
	pLock sync.RWMutex

	p2pClient  P2pClient
	storageRpc StorageRpc

	taskChan      chan *mtypes.SyncDataRequest
	taskHeight    uint64
	taskInProcess map[string]*mtypes.SyncDataRequest
	tLock         sync.Mutex

	dataChan     chan *mtypes.SyncDataResponse
	dataHeight   uint64
	waitForWrite map[uint64][]*mtypes.SyncDataRequest

	writeChan chan []*mtypes.SyncDataRequest
}

func NewSyncClient(concurrency int, groupId string) actor.IWorkerEx {
	cli := &SyncClient{
		peers:         make(map[string]*mtypes.SyncStatus),
		taskChan:      make(chan *mtypes.SyncDataRequest, 1000),
		taskInProcess: make(map[string]*mtypes.SyncDataRequest),
		dataChan:      make(chan *mtypes.SyncDataResponse, 1000),
		waitForWrite:  make(map[uint64][]*mtypes.SyncDataRequest),
		writeChan:     make(chan []*mtypes.SyncDataRequest, 1000),
		p2pClient:     p2p.NewP2pClient(concurrency, "p2p.client").(P2pClient),
		storageRpc:    NewDefaultStorageRpc(),
	}
	cli.Set(concurrency, groupId)
	return cli
}

func (cli *SyncClient) TestOnlySetP2pClient(p2pClient P2pClient) {
	cli.p2pClient = p2pClient
}

func (srv *SyncClient) TestOnlySetStorageRpc(storageRpc StorageRpc) {
	srv.storageRpc = storageRpc
}

func (cli *SyncClient) Inputs() ([]string, bool) {
	return []string{
		actor.MsgStateSyncStart,
		actor.MsgP2pResponse,
	}, false
}

func (cli *SyncClient) Outputs() map[string]int {
	return map[string]int{
		actor.MsgStateSyncDone: 1,
	}
}

func (cli *SyncClient) OnStart() {
	cli.p2pClient.OnConnClosed(func(id string) {
		cli.pLock.Lock()
		defer cli.pLock.Unlock()
		delete(cli.peers, id)
	})

	go func() {
		for {
			fmt.Printf("[SyncClient.OnStart] broadcast sync status request\n")
			cli.p2pClient.Broadcast(&actor.Message{
				Name: actor.MsgSyncStatusRequest,
			})
			time.Sleep(10 * time.Second)
		}
	}()

	go cli.timeoutTaskRecyclingRoutine()
}

func (cli *SyncClient) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgStateSyncStart:
		fmt.Printf("[SyncClient.OnMessageArrived] MsgStateSyncStart: target = %d\n", msg.Data.(uint64))
		status := cli.storageRpc.GetSyncStatus()
		cli.taskHeight = status.Height
		cli.dataHeight = status.Height

		target := msg.Data.(uint64)
		// Get peers status.
		go func() {
			// Find one eligible peer.
			var status *mtypes.SyncStatus
			for {
				status = cli.getPeerAbove(target)
				if status != nil {
					break
				}
				time.Sleep(3 * time.Second)
			}

			if cli.dataHeight < status.SyncPoint {
				cli.dataHeight = 0
			}

			go cli.taskCreationRoutine(target, status.SyncPoint)
			go cli.taskSendingRoutine()
			go cli.dataReorderingRoutine(target)
			go cli.dataWritingRoutine()
		}()
	case actor.MsgP2pResponse:
		if msg.Data == nil {
			fmt.Printf("[SyncClient.OnMessageArrived] Receive MsgP2pResponse with nil data.\n")
			return nil
		}
		p2pMessage := msg.Data.(*p2p.P2pMessage)
		msg = p2pMessage.Message
		switch msg.Name {
		case actor.MsgSyncStatusResponse:
			fmt.Printf("[SyncClient.OnMessageArrived] MsgSyncStatusResponse: %v\n", msg.Data.(*mtypes.SyncStatus))
			cli.pLock.Lock()
			defer cli.pLock.Unlock()
			ss := msg.Data.(*mtypes.SyncStatus)
			if _, ok := cli.peers[ss.Id]; !ok {
				cli.peers[ss.Id] = ss
			} else {
				cli.peers[ss.Id].SyncPoint = ss.SyncPoint
				cli.peers[ss.Id].Height = ss.Height
			}
		case actor.MsgSyncPointResponse:
			fmt.Printf("[SyncClient.OnMessageArrived] MsgSyncPointResponse: %v\n", msg.Data.(*mtypes.SyncPoint))
			sp := msg.Data.(*mtypes.SyncPoint)
			// Save parent info, but leave the Height at the beginning of the sync point.
			// The Height will be updated to the end of the sync point when the sync point
			// was applied to local url store.
			var na int
			intf.Router.Call("statestore", "Save", &storage.State{
				Height:     sp.From,
				ParentHash: sp.Parent.ParentHash,
				ParentRoot: sp.Parent.ParentRoot,
			}, &na)

			states := sp.SchdStates.([]storage.SchdState)
			for _, state := range states {
				intf.Router.Call("schdstore", "DirectWrite", &state, &na)
			}

			// TODO: validate the sync point.
			go func() {
				for i := range sp.Slices {
					cli.taskChan <- &mtypes.SyncDataRequest{
						From:  sp.From,
						To:    sp.To,
						Slice: i,
					}
				}
			}()
		case actor.MsgSyncDataResponse:
			fmt.Printf("[SyncClient.OnMessageArrived] MsgSyncDataResponse: %v\n", msg.Data.(*mtypes.SyncDataResponse).SyncDataRequest)
			data := msg.Data.(*mtypes.SyncDataResponse)
			cli.tLock.Lock()
			delete(cli.taskInProcess, data.ID())
			cli.tLock.Unlock()
			cli.setPeerIdle(data.ID())
			cli.dataChan <- data
		}
	}
	return nil
}

func (cli *SyncClient) getPeerAbove(height uint64) *mtypes.SyncStatus {
	cli.pLock.RLock()
	defer cli.pLock.RUnlock()
	fmt.Printf("[SyncClient.getPeerAbove] height = %d\n", height)
	for _, status := range cli.peers {
		fmt.Printf("[SyncClient.getPeerAbove] peer status = %v\n", status)
		if status.RequestId == "" && status.Height >= height {
			return status
		}
	}
	return nil
}

func (cli *SyncClient) getNextTask(target, syncPoint uint64) *mtypes.SyncDataRequest {
	if syncPoint > cli.taskHeight {
		cli.taskHeight = syncPoint
		return &mtypes.SyncDataRequest{
			From:  0,
			To:    syncPoint,
			Slice: -1,
		}
	}

	if target > cli.taskHeight {
		cli.taskHeight++
		return &mtypes.SyncDataRequest{
			From:  cli.taskHeight - 1,
			To:    cli.taskHeight,
			Slice: 0,
		}
	}

	return nil
}

func (cli *SyncClient) setPeerIdle(requestId string) {
	cli.pLock.RLock()
	defer cli.pLock.RUnlock()
	for _, status := range cli.peers {
		if status.RequestId == requestId {
			status.RequestId = ""
			return
		}
	}
}

func (cli *SyncClient) taskCreationRoutine(target, syncPoint uint64) {
	for {
		task := cli.getNextTask(target, syncPoint)
		if task != nil {
			cli.taskChan <- task
		} else {
			// End routine.
			return
		}
	}
}

func (cli *SyncClient) taskSendingRoutine() {
	for task := range cli.taskChan {
		if task == nil {
			return
		}

		status := cli.getPeerAbove(task.To)
		if status == nil {
			time.Sleep(10 * time.Microsecond)
			cli.taskChan <- task
			continue
		}

		if task.To-task.From > 1 && task.Slice == -1 {
			cli.p2pClient.Request(status.Id, &actor.Message{
				Name: actor.MsgSyncPointRequest,
				Data: task.To,
			})
		} else {
			cli.p2pClient.Request(status.Id, &actor.Message{
				Name: actor.MsgSyncDataRequest,
				Data: task,
			})
			status.RequestId = task.ID()
			cli.tLock.Lock()
			cli.taskInProcess[task.ID()] = &mtypes.SyncDataRequest{
				From:    task.From,
				To:      task.To,
				Slice:   task.Slice,
				StartAt: time.Now(),
			}
			cli.tLock.Unlock()
		}
	}
}

func (cli *SyncClient) dataReorderingRoutine(target uint64) {
	for data := range cli.dataChan {
		cli.storageRpc.WriteSlice(data)
		if data.To-data.From > 1 {
			if _, ok := cli.waitForWrite[data.From]; !ok {
				cli.waitForWrite[data.From] = make([]*mtypes.SyncDataRequest, mtypes.SlicePerSyncPoint)
				cli.waitForWrite[data.From][data.Slice] = &data.SyncDataRequest
			} else {
				cli.waitForWrite[data.From][data.Slice] = &data.SyncDataRequest
			}
		} else {
			cli.waitForWrite[data.From] = []*mtypes.SyncDataRequest{&data.SyncDataRequest}
		}

		for {
			if _, ok := cli.waitForWrite[cli.dataHeight]; ok {
				completed := true
				for i := range cli.waitForWrite[cli.dataHeight] {
					if cli.waitForWrite[cli.dataHeight][i] == nil {
						completed = false
						break
					}
				}

				if completed {
					slices := cli.waitForWrite[cli.dataHeight]
					cli.writeChan <- slices
					delete(cli.waitForWrite, cli.dataHeight)
					cli.dataHeight = slices[0].To

					// Send signal to all routines.
					if cli.dataHeight >= target {
						cli.taskChan <- nil
						cli.writeChan <- nil
						return
					}
				} else {
					break
				}
			} else {
				break
			}
		}
	}
}

func (cli *SyncClient) dataWritingRoutine() {
	startAt := time.Now()
	for slices := range cli.writeChan {
		if slices == nil {
			cli.storageRpc.RewriteMeta()
			fmt.Printf("SyncClient[%s]: data sync done [%v]\n", cli.Groupid, time.Since(startAt))
			cli.MsgBroker.Send(actor.MsgStateSyncDone, nil)
			return
		}

		cli.storageRpc.ApplyData(slices[0])
	}
}

func (cli *SyncClient) timeoutTaskRecyclingRoutine() {
	for {
		cli.tLock.Lock()
		for id, request := range cli.taskInProcess {
			if time.Since(request.StartAt) > 30*time.Second {
				request.StartAt = time.Now()
				cli.taskChan <- request
			}
			cli.setPeerIdle(id)
		}
		cli.tLock.Unlock()

		time.Sleep(100 * time.Millisecond)
	}
}
