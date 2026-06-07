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
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"

	"github.com/arcology-network/common-lib/common"
	badgerpk "github.com/arcology-network/common-lib/storage/badger"
	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/main/components/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"

	scommon "github.com/arcology-network/streamer/common"
)

var (
	ssStore *StateSyncStore
)

type KvDB interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

const (
	s3StateUninit = iota
	s3StateSchdState
	s3StateUrlUpdate
	s3StateAcctHash
	s3StateParentInfo
)

type StateSyncStore struct {
	state      int
	sliceDB    KvDB
	spDB       *badgerpk.ParaBadgerDB
	spInterval uint64
	status     *mtypes.SyncStatus
	sp         *mtypes.SyncPoint

	// Bufferred data
	urlUpdate *storage.UrlUpdate
	hash      *evmCommon.Hash
	// schdState *mtypes.SchdState

	//
	parent *mtypes.ParentInfo
	height uint64
	// states []mtypes.SchdState
}

func NewStateSyncStore() actor.Business {
	ssStore = &StateSyncStore{
		state: s3StateUninit,
	}
	return ssStore
}

// func TestOnlyNewStateSyncStore(concurrency int, groupId string) actor.IWorkerEx {
// 	store := &StateSyncStore{}
// 	store.Set(concurrency, groupId)
// 	return store
// }

// func (store *StateSyncStore) TestOnlyGetSyncPointDB() *badgerpk.ParaBadgerDB {
// 	return store.spDB
// }

func (store *StateSyncStore) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgParentInfo,
		// scommon.MsgSchdState,
		scommon.MsgUrlUpdate,
		scommon.MsgAcctHash,
	}, false
}

func (store *StateSyncStore) Outputs() map[string]int {
	return map[string]int{}
}

func (store *StateSyncStore) RpcConfig() (string, int) {
	return "statesyncstore", 20
}

func (store *StateSyncStore) Config(params map[string]interface{}) {
	store.sliceDB = transactional.NewSimpleFileDB(params["slice_db_root"].(string))
	store.spDB = badgerpk.NewParaBadgerDB(params["sync_point_root"].(string), common.Remainder)
	store.spInterval = uint64(params["sync_point_interval"].(int))
}

func (store *StateSyncStore) GetFSMRules() map[int]actor.FSMRule {
	return map[int]actor.FSMRule{
		s3StateUninit: {Accept: []string{
			scommon.MsgParentInfo,
		}},
		// s3StateSchdState: {Accept: []string{
		// 	scommon.MsgSchdState,
		// }},
		s3StateUrlUpdate: {Accept: []string{
			scommon.MsgUrlUpdate,
		}},
		s3StateAcctHash: {Accept: []string{
			scommon.MsgAcctHash,
		}},
		s3StateParentInfo: {Accept: []string{
			scommon.MsgParentInfo,
		}},
	}
}

func (store *StateSyncStore) GetCurrentState() int {
	return store.state
}

func (store *StateSyncStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgParentInfo, store.receivedParentInfo)
	// reg.Register(scommon.MsgSchdState, store.receivedSchdState)
	reg.Register(scommon.MsgUrlUpdate, store.receivedUrlUpdate)
	reg.Register(scommon.MsgAcctHash, store.receivedAcctHash)
	reg.Register("GetSyncStatus", store.GetSyncStatus)
	reg.Register("GetSyncPoint", store.GetSyncPoint)
	reg.Register("InitSyncPoint", store.InitSyncPoint)
	reg.Register("LoadSchd", store.LoadSchd)
	reg.Register("setSyncPoint_back", store.setSyncPoint_back)
	reg.Register("WriteSlice", store.WriteSlice)
	reg.Register("ReadSlice", store.ReadSlice)
	reg.Register("SetSyncPoint", store.SetSyncPoint)
	reg.Register("SetSyncStatus", store.SetSyncStatus)
	reg.Register("makeSyncPointLoad", store.makeSyncPointLoad)
}

func (store *StateSyncStore) receivedAcctHash(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	store.hash = msg.Data.(*evmCommon.Hash)
	store.state = s3StateParentInfo
	return nil
}

func (store *StateSyncStore) receivedUrlUpdate(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	store.urlUpdate = msg.Data.(*storage.UrlUpdate)
	store.state = s3StateAcctHash
	// Debug
	keySize := 0
	for _, k := range store.urlUpdate.Keys {
		keySize += len(k)
	}
	valueSize := 0
	for _, v := range store.urlUpdate.EncodedValues {
		valueSize += len(v)
	}
	ctx.ExecCtx.LogDebug(fmt.Sprintf("[StateSyncStore.OnMessageArrived] MsgUrlUpdate received, len(keys) = %d, len(values) = %d, total key size = %d, total value size = %d", len(store.urlUpdate.Keys), len(store.urlUpdate.EncodedValues), keySize, valueSize))
	return nil
}

func (store *StateSyncStore) receivedSchdState(ctx *actor.ActionContext) error {
	// msg := ctx.Messages[0]
	// store.schdState = msg.Data.(*mtypes.SchdState)
	store.state = s3StateUrlUpdate
	return nil
}

func (store *StateSyncStore) receivedParentInfo(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	switch store.state {
	case s3StateUninit:
		ctx.ExecCtx.LogDebug("[StateSyncStore.OnMessageArrived] Ignore the first ParentInfo")
		store.state = s3StateSchdState
	case s3StateParentInfo:
		parent := msg.Data.(*mtypes.ParentInfo)

		store.WriteSliceInner(&mtypes.SyncDataResponse{
			SyncDataRequest: mtypes.SyncDataRequest{
				From:  msg.Height - 1,
				To:    msg.Height,
				Slice: 0,
			},
			Hash:   store.hash.Bytes(),
			Data:   store.encode(store.urlUpdate),
			Parent: parent,
			// SchdStates: store.schdState,
		})
		// var na int
		// store.WriteSlice(context.Background(), &mtypes.SyncDataResponse{
		// 	SyncDataRequest: mtypes.SyncDataRequest{
		// 		From:  msg.Height - 1,
		// 		To:    msg.Height,
		// 		Slice: 0,
		// 	},
		// 	Hash:       store.hash.Bytes(),
		// 	Data:       store.encode(store.urlUpdate),
		// 	Parent:     parent,
		// 	SchdStates: store.schdState,
		// }, &na)

		status := *store.getSyncStatus()
		status.Height = msg.Height
		store.setSyncStatus(&status)

		if msg.Height%store.spInterval == 0 && msg.Height != 0 {
			// Apply blocks from current sync point to new sync point.
			store.makeSyncPoint(ctx, status.SyncPoint, msg.Height)
		} else {
			store.changeStateTos3StateSchdState()
		}

	}
	return nil
}

func (store *StateSyncStore) changeStateTos3StateSchdState() {
	store.state = s3StateSchdState
}

func (store *StateSyncStore) setSyncStatus(status *mtypes.SyncStatus) error {
	store.status = status
	return store.sliceDB.Set("syncstatus", store.encode(status))
}

func (store *StateSyncStore) SetSyncStatus(ctx *actor.ActionContext) error {
	status := ctx.RPC.Request.(*mtypes.SyncStatus)
	store.setSyncStatus(status)
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (store *StateSyncStore) getSyncStatus() *mtypes.SyncStatus {
	if store.status == nil {
		store.status = &mtypes.SyncStatus{}
		bs, err := store.sliceDB.Get("syncstatus")
		if err == nil {
			store.decode(bs, store.status)
		}
	}
	return store.status
}

func (store *StateSyncStore) GetSyncStatus(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", store.getSyncStatus())
	return nil
}

func (store *StateSyncStore) setSyncPoint(sp *mtypes.SyncPoint) error {
	return store.sliceDB.Set("syncpoint", store.encode(sp))
}

func (store *StateSyncStore) SetSyncPoint(ctx *actor.ActionContext) error {
	sp := ctx.RPC.Request.(*mtypes.SyncPoint)
	store.sp = sp
	store.setSyncPoint(sp)
	ctx.ExecCtx.SendRpcResponse("", "")
	return nil
}

func (store *StateSyncStore) getSyncPoint() *mtypes.SyncPoint {
	if store.sp == nil {
		store.sp = &mtypes.SyncPoint{}
		bs, err := store.sliceDB.Get("syncpoint")
		if err == nil {
			store.decode(bs, store.sp)
		}
	}
	return store.sp
}

func (store *StateSyncStore) GetSyncPoint(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(uint64)
	sp := store.getSyncPoint()
	if sp.To != height {
		ctx.ExecCtx.SendRpcResponse("syncpoint not found", nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", sp)
	}
	return nil
}

func (store *StateSyncStore) setSyncPoint_back(ctx *actor.ActionContext) error {
	// store.states = ctx.RPC.Request.([]mtypes.SchdState)

	// end := 0
	// for i, state := range store.states {
	// 	if state.Height > store.height {
	// 		end = i
	// 		break
	// 	}
	// }

	// // TODO: Set slice hashes.
	// if err := store.setSyncPoint(&mtypes.SyncPoint{
	// 	From:       0,
	// 	To:         store.height,
	// 	Slices:     make([]evmCommon.Hash, mtypes.SlicePerSyncPoint),
	// 	Parent:     store.parent,
	// 	SchdStates: store.states[:end],
	// }); err != nil {
	// 	// ctx.ExecCtx.EndCasecade()
	// 	ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	// 	return err
	// }
	// ctx.ExecCtx.EndCasecade()
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (store *StateSyncStore) LoadSchd(ctx *actor.ActionContext) error {
	store.parent = ctx.RPC.Request.(*mtypes.ParentInfo)
	ctx.ExecCtx.InvokeRPC("schdstore", "Load", "", "setSyncPoint_back")
	return nil
}

func (store *StateSyncStore) InitSyncPoint(ctx *actor.ActionContext) error {
	store.height = ctx.RPC.Request.(uint64)
	count := 0
	for i := 0; i < mtypes.SlicePerSyncPoint; i++ {
		if response, err := store.readSliceFromKvDB(&mtypes.SyncDataRequest{
			From:  0,
			To:    store.height,
			Slice: i,
		}); err != nil {
			ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
			return err
		} else {
			var urlUpdate storage.UrlUpdate
			gob.NewDecoder(bytes.NewBuffer(response.Data)).Decode(&urlUpdate)
			store.spDB.SetBatch(urlUpdate.Keys, urlUpdate.EncodedValues)
			count += len(urlUpdate.Keys)
		}
	}
	ctx.ExecCtx.LogDebug(fmt.Sprintf("StateSyncStore.InitSyncPoint, update %d keys", count))

	// ctx.ExecCtx.StartCasecade()
	ctx.ExecCtx.InvokeRPC("statestore", "GetParentInfo", "", "LoadSchd")
	return nil

}

func (store *StateSyncStore) WriteSlice(ctx *actor.ActionContext) error {
	slice := ctx.RPC.Request.(*mtypes.SyncDataResponse)
	err := store.WriteSliceInner(slice)
	// store.sliceDB.Set(store.sliceKey(&slice.SyncDataRequest), store.encode(slice))
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nil)
	}
	return nil
}

func (store *StateSyncStore) WriteSliceInner(resp *mtypes.SyncDataResponse) error {
	return store.sliceDB.Set(store.sliceKey(&resp.SyncDataRequest), store.encode(resp))
}

func (store *StateSyncStore) deleteSlice(slice *mtypes.SyncDataRequest) error {
	return store.sliceDB.Delete(store.sliceKey(slice))
}

func (store *StateSyncStore) readSliceFromSyncPointDB(request *mtypes.SyncDataRequest) (*mtypes.SyncDataResponse, error) {
	// TODO: check request.To & request.From
	keys, values, err := store.spDB.Query(fmt.Sprintf("%s%02x", RootPrefix, []byte{byte(request.Slice)}), nil)
	for i := range err {
		if err[i] != nil {
			return nil, err[i]
		}
	}

	// TODO: Calculate slice hash.
	response := &mtypes.SyncDataResponse{
		SyncDataRequest: *request,
		Data: store.encode(&storage.UrlUpdate{
			Keys:          keys,
			EncodedValues: values,
		}),
	}
	fmt.Printf("StateSyncStore.ReadSlice, load %d keys\n", len(keys))
	return response, nil
}

func (store *StateSyncStore) readSliceFromKvDB(request *mtypes.SyncDataRequest) (*mtypes.SyncDataResponse, error) {
	bs, err := store.sliceDB.Get(store.sliceKey(request))
	if err != nil {
		return nil, err
	}

	var response mtypes.SyncDataResponse
	store.decode(bs, &response)
	if response.To != request.To {
		return nil, errors.New("slice not found")
	}
	return &response, nil
}

func (store *StateSyncStore) ReadSlice(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.SyncDataRequest)
	// var resp *mtypes.SyncDataResponse
	// var err error
	// if request.To-request.From > 1 {
	// 	resp, err = store.readSliceFromSyncPointDB(request)
	// } else {
	// 	resp, err = store.readSliceFromKvDB(request)
	// }

	resp, err := store.ReadSliceInner(request)

	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return err
	}
	ctx.ExecCtx.SendRpcResponse("", resp)
	return nil
}

func (store *StateSyncStore) ReadSliceInner(request *mtypes.SyncDataRequest) (*mtypes.SyncDataResponse, error) {
	var resp *mtypes.SyncDataResponse
	var err error
	if request.To-request.From > 1 {
		resp, err = store.readSliceFromSyncPointDB(request)
	} else {
		resp, err = store.readSliceFromKvDB(request)
	}
	return resp, err
}

func (store *StateSyncStore) encode(obj interface{}) []byte {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(obj)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (store *StateSyncStore) decode(bs []byte, obj interface{}) {
	err := gob.NewDecoder(bytes.NewBuffer(bs)).Decode(obj)
	if err != nil {
		panic(err)
	}
}

func (store *StateSyncStore) sliceKey(slice *mtypes.SyncDataRequest) string {
	return fmt.Sprintf("%016x-%04x", slice.From, slice.Slice)
}

func (store *StateSyncStore) makeSyncPoint(ctx *actor.ActionContext, from, to uint64) {
	status := *store.getSyncStatus()
	// Disable sync point.
	status.SyncPoint = 0
	store.setSyncStatus(&status)

	var parent *mtypes.ParentInfo
	for i := from; i < to; i++ {

		response, err := store.ReadSliceInner(&mtypes.SyncDataRequest{
			From:  i,
			To:    i + 1,
			Slice: 0,
		})

		// var response mtypes.SyncDataResponse
		// err := store.ReadSlice(context.Background(), &mtypes.SyncDataRequest{
		// 	From:  i,
		// 	To:    i + 1,
		// 	Slice: 0,
		// }, &response)
		if err != nil {
			panic(err)
		}
		parent = response.Parent

		var urlUpdate storage.UrlUpdate
		gob.NewDecoder(bytes.NewBuffer(response.Data)).Decode(&urlUpdate)
		store.spDB.SetBatch(urlUpdate.Keys, urlUpdate.EncodedValues)

		err = store.deleteSlice(&mtypes.SyncDataRequest{
			From:  i,
			To:    i + 1,
			Slice: 0,
		})
		if err != nil {
			fmt.Printf("[StateSyncStore] deleteSlice(%d) failed, err = %v\n", i, err)
		}
	}
	store.parent = parent
	store.height = to
	store.status = &status
	ctx.ExecCtx.InvokeRPC("schdstore", "Load", "", "makeSyncPointLoad")
	return

	// var states []mtypes.SchdState
	// var na int
	// intf.Router.Call("schdstore", "Load", &na, &states)
	// end := 0
	// for i, state := range states {
	// 	if state.Height > to {
	// 		end = i
	// 		break
	// 	}
	// }

	// // TODO: Set slice hashes.
	// store.setSyncPoint(&mtypes.SyncPoint{
	// 	From:       0,
	// 	To:         to,
	// 	Slices:     make([]evmCommon.Hash, mtypes.SlicePerSyncPoint),
	// 	Parent:     parent,
	// 	SchdStates: states[:end],
	// })

	// // Enable sync point.
	// status.SyncPoint = to
	// store.setSyncStatus(&status)
}

func (store *StateSyncStore) makeSyncPointLoad(ctx *actor.ActionContext) error {
	// states := ctx.RPC.Request.([]mtypes.SchdState)
	// end := 0
	// for i, state := range states {
	// 	if state.Height > store.height {
	// 		end = i
	// 		break
	// 	}
	// }
	// // TODO: Set slice hashes.
	// store.setSyncPoint(&mtypes.SyncPoint{
	// 	From:       0,
	// 	To:         store.height,
	// 	Slices:     make([]evmCommon.Hash, mtypes.SlicePerSyncPoint),
	// 	Parent:     store.parent,
	// 	SchdStates: states[:end],
	// })

	// Enable sync point.
	store.status.SyncPoint = store.height
	store.setSyncStatus(store.status)

	store.changeStateTos3StateSchdState()
	return nil
}
