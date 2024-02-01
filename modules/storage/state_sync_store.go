package storage

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"

	"github.com/arcology-network/common-lib/common"
	badgerpk "github.com/arcology-network/common-lib/storage/badger"
	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/main/components/storage"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

var (
	ssStore *StateSyncStore
	initS3  sync.Once
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
	actor.WorkerThread

	state      int
	sliceDB    KvDB
	spDB       *badgerpk.ParaBadgerDB
	spInterval uint64
	status     *mtypes.SyncStatus
	sp         *mtypes.SyncPoint

	// Bufferred data
	urlUpdate *storage.UrlUpdate
	hash      *evmCommon.Hash
	schdState *SchdState
}

func NewStateSyncStore(concurrency int, groupId string) actor.IWorkerEx {
	initS3.Do(func() {
		ssStore = &StateSyncStore{
			state: s3StateUninit,
		}
		ssStore.Set(concurrency, groupId)
	})
	return ssStore
}

func TestOnlyNewStateSyncStore(concurrency int, groupId string) actor.IWorkerEx {
	store := &StateSyncStore{}
	store.Set(concurrency, groupId)
	return store
}

func (store *StateSyncStore) TestOnlyGetSyncPointDB() *badgerpk.ParaBadgerDB {
	return store.spDB
}

func (store *StateSyncStore) Inputs() ([]string, bool) {
	return []string{
		actor.MsgParentInfo,
		actor.MsgSchdState,
		actor.MsgUrlUpdate,
		actor.MsgAcctHash,
	}, false
}

func (store *StateSyncStore) Outputs() map[string]int {
	return map[string]int{}
}

func (store *StateSyncStore) Config(params map[string]interface{}) {
	store.sliceDB = transactional.NewSimpleFileDB(params["slice_db_root"].(string))
	store.spDB = badgerpk.NewParaBadgerDB(params["sync_point_root"].(string), common.Remainder)
	store.spInterval = uint64(params["sync_point_interval"].(float64))
}

func (store *StateSyncStore) OnStart() {}

func (store *StateSyncStore) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch store.state {
	case s3StateUninit:
		fmt.Printf("[StateSyncStore.OnMessageArrived] Ignore the first ParentInfo\n")
		store.state = s3StateSchdState
	case s3StateSchdState:
		store.schdState = msg.Data.(*SchdState)
		store.state = s3StateUrlUpdate
	case s3StateUrlUpdate:
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
		fmt.Printf(
			"[StateSyncStore.OnMessageArrived] MsgUrlUpdate received, len(keys) = %d, len(values) = %d, total key size = %d, total value size = %d\n",
			len(store.urlUpdate.Keys),
			len(store.urlUpdate.EncodedValues),
			keySize, valueSize)
	case s3StateAcctHash:
		store.hash = msg.Data.(*evmCommon.Hash)
		store.state = s3StateParentInfo
	case s3StateParentInfo:
		parent := msg.Data.(*mtypes.ParentInfo)
		var na int
		store.WriteSlice(context.Background(), &mtypes.SyncDataResponse{
			SyncDataRequest: mtypes.SyncDataRequest{
				From:  msg.Height - 1,
				To:    msg.Height,
				Slice: 0,
			},
			Hash:       store.hash.Bytes(),
			Data:       store.encode(store.urlUpdate),
			Parent:     parent,
			SchdStates: store.schdState,
		}, &na)

		status := *store.getSyncStatus()
		status.Height = msg.Height
		store.setSyncStatus(&status)

		if msg.Height%store.spInterval == 0 && msg.Height != 0 {
			// Apply blocks from current sync point to new sync point.
			store.makeSyncPoint(status.SyncPoint, msg.Height)
		}
		store.state = s3StateSchdState
	}
	return nil
}

func (store *StateSyncStore) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		s3StateUninit:     {actor.MsgParentInfo},
		s3StateSchdState:  {actor.MsgSchdState},
		s3StateUrlUpdate:  {actor.MsgUrlUpdate},
		s3StateAcctHash:   {actor.MsgAcctHash},
		s3StateParentInfo: {actor.MsgParentInfo},
	}
}

func (store *StateSyncStore) GetCurrentState() int {
	return store.state
}

func (store *StateSyncStore) setSyncStatus(status *mtypes.SyncStatus) error {
	store.status = status
	return store.sliceDB.Set("syncstatus", store.encode(status))
}

func (store *StateSyncStore) SetSyncStatus(ctx context.Context, status *mtypes.SyncStatus, _ *int) error {
	return store.setSyncStatus(status)
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

func (store *StateSyncStore) GetSyncStatus(ctx context.Context, _ *int, status *mtypes.SyncStatus) error {
	*status = *store.getSyncStatus()
	return nil
}

func (store *StateSyncStore) setSyncPoint(sp *mtypes.SyncPoint) error {
	return store.sliceDB.Set("syncpoint", store.encode(sp))
}

func (store *StateSyncStore) SetSyncPoint(ctx context.Context, sp *mtypes.SyncPoint, _ *int) error {
	store.sp = sp
	return store.setSyncPoint(sp)
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

func (store *StateSyncStore) GetSyncPoint(ctx context.Context, height *uint64, sp *mtypes.SyncPoint) error {
	*sp = *store.getSyncPoint()
	if sp.To != *height {
		return errors.New("syncpoint not found")
	}
	return nil
}

func (store *StateSyncStore) InitSyncPoint(ctx context.Context, to *uint64, sp *mtypes.SyncPoint) error {
	count := 0
	for i := 0; i < mtypes.SlicePerSyncPoint; i++ {
		if response, err := store.readSliceFromKvDB(&mtypes.SyncDataRequest{
			From:  0,
			To:    *to,
			Slice: i,
		}); err != nil {
			return err
		} else {
			var urlUpdate storage.UrlUpdate
			gob.NewDecoder(bytes.NewBuffer(response.Data)).Decode(&urlUpdate)
			store.spDB.BatchSet(urlUpdate.Keys, urlUpdate.EncodedValues)
			count += len(urlUpdate.Keys)
		}
	}
	fmt.Printf("StateSyncStore.InitSyncPoint, update %d keys\n", count)

	var na int
	var parent mtypes.ParentInfo
	intf.Router.Call("statestore", "GetParentInfo", &na, parent)

	var states []SchdState
	intf.Router.Call("schdstore", "Load", &na, &states)
	end := 0
	for i, state := range states {
		if state.Height > *to {
			end = i
			break
		}
	}

	// TODO: Set slice hashes.
	if err := store.setSyncPoint(&mtypes.SyncPoint{
		From:       0,
		To:         *to,
		Slices:     make([]evmCommon.Hash, mtypes.SlicePerSyncPoint),
		Parent:     &parent,
		SchdStates: states[:end],
	}); err != nil {
		return err
	}
	return nil
}

func (store *StateSyncStore) WriteSlice(ctx context.Context, slice *mtypes.SyncDataResponse, _ *int) error {
	return store.sliceDB.Set(store.sliceKey(&slice.SyncDataRequest), store.encode(slice))
}

func (store *StateSyncStore) deleteSlice(slice *mtypes.SyncDataRequest) error {
	return store.sliceDB.Delete(store.sliceKey(slice))
}

func (store *StateSyncStore) readSliceFromSyncPointDB(request *mtypes.SyncDataRequest) (*mtypes.SyncDataResponse, error) {
	// TODO: check request.To & request.From
	keys, values, err := store.spDB.Query(fmt.Sprintf("%s%02x", RootPrefix, []byte{byte(request.Slice)}), nil)
	if err != nil {
		return nil, err
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

func (store *StateSyncStore) ReadSlice(ctx context.Context, request *mtypes.SyncDataRequest, response *mtypes.SyncDataResponse) error {
	var resp *mtypes.SyncDataResponse
	var err error
	if request.To-request.From > 1 {
		resp, err = store.readSliceFromSyncPointDB(request)
	} else {
		resp, err = store.readSliceFromKvDB(request)
	}

	if err != nil {
		return err
	}
	*response = *resp
	return nil
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

func (store *StateSyncStore) makeSyncPoint(from, to uint64) {
	status := *store.getSyncStatus()
	// Disable sync point.
	status.SyncPoint = 0
	store.setSyncStatus(&status)

	var parent *mtypes.ParentInfo
	for i := from; i < to; i++ {
		var response mtypes.SyncDataResponse
		err := store.ReadSlice(context.Background(), &mtypes.SyncDataRequest{
			From:  i,
			To:    i + 1,
			Slice: 0,
		}, &response)
		if err != nil {
			panic(err)
		}
		parent = response.Parent

		var urlUpdate storage.UrlUpdate
		gob.NewDecoder(bytes.NewBuffer(response.Data)).Decode(&urlUpdate)
		store.spDB.BatchSet(urlUpdate.Keys, urlUpdate.EncodedValues)

		err = store.deleteSlice(&mtypes.SyncDataRequest{
			From:  i,
			To:    i + 1,
			Slice: 0,
		})
		if err != nil {
			fmt.Printf("[StateSyncStore] deleteSlice(%d) failed, err = %v\n", i, err)
		}
	}

	var states []SchdState
	var na int
	intf.Router.Call("schdstore", "Load", &na, &states)
	end := 0
	for i, state := range states {
		if state.Height > to {
			end = i
			break
		}
	}

	// TODO: Set slice hashes.
	store.setSyncPoint(&mtypes.SyncPoint{
		From:       0,
		To:         to,
		Slices:     make([]evmCommon.Hash, mtypes.SlicePerSyncPoint),
		Parent:     parent,
		SchdStates: states[:end],
	})

	// Enable sync point.
	status.SyncPoint = to
	store.setSyncStatus(&status)
}
