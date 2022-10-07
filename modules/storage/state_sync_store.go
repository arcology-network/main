package storage

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"sync"

	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/cachedstorage"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	intf "github.com/HPISTechnologies/component-lib/interface"
	"github.com/HPISTechnologies/component-lib/storage"
	urlcmn "github.com/HPISTechnologies/concurrenturl/v2/common"
)

var (
	ssStore *StateSyncStore
	initS3  sync.Once
	urlRoot = urlcmn.NewPlatform().Eth10Account()
)

type KvDB interface {
	Set(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

const (
	s3StateUninit = iota
	s3StateWaitingUpdates
	s3StateWaitingParentInfo
)

type StateSyncStore struct {
	actor.WorkerThread

	state      int
	sliceDB    KvDB
	spDB       *cachedstorage.ParaBadgerDB
	spInterval uint64
	status     *cmntyp.SyncStatus
	sp         *cmntyp.SyncPoint

	// Bufferred data
	urlUpdate *storage.UrlUpdate
	hash      *ethcmn.Hash
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

func (store *StateSyncStore) TestOnlyGetSyncPointDB() *cachedstorage.ParaBadgerDB {
	return store.spDB
}

func (store *StateSyncStore) Inputs() ([]string, bool) {
	return []string{
		actor.MsgParentInfo,
		actor.CombinedName(actor.MsgUrlUpdate, actor.MsgAcctHash),
	}, false
}

func (store *StateSyncStore) Outputs() map[string]int {
	return map[string]int{}
}

func (store *StateSyncStore) Config(params map[string]interface{}) {
	store.sliceDB = NewSimpleFileDB(params["slice_db_root"].(string))
	store.spDB = cachedstorage.NewParaBadgerDB(params["sync_point_root"].(string))
	store.spInterval = uint64(params["sync_point_interval"].(float64))
}

func (store *StateSyncStore) OnStart() {}

func (store *StateSyncStore) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch msg.Name {
	case actor.MsgParentInfo:
		// Ignore the first ParentInfo, it is for the genesis block.
		if store.state == s3StateUninit {
			fmt.Printf("[StateSyncStore.OnMessageArrived] Ignore the first ParentInfo\n")
			store.state = s3StateWaitingUpdates
		} else {
			parent := msg.Data.(*cmntyp.ParentInfo)
			var na int
			store.WriteSlice(context.Background(), &cmntyp.SyncDataResponse{
				SyncDataRequest: cmntyp.SyncDataRequest{
					From:  msg.Height - 1,
					To:    msg.Height,
					Slice: 0,
				},
				Hash:   store.hash.Bytes(),
				Data:   store.encode(store.urlUpdate),
				Parent: parent,
			}, &na)

			status := *store.getSyncStatus()
			status.Height = msg.Height
			store.setSyncStatus(&status)

			if msg.Height%store.spInterval == 0 && msg.Height != 0 {
				// Apply blocks from current sync point to new sync point.
				store.makeSyncPoint(status.SyncPoint, msg.Height)
			}
			store.state = s3StateWaitingUpdates
		}
	case actor.CombinedName(actor.MsgUrlUpdate, actor.MsgAcctHash):
		combined := msg.Data.(*actor.CombinerElements)
		store.urlUpdate = combined.Get(actor.MsgUrlUpdate).Data.(*storage.UrlUpdate)
		store.hash = combined.Get(actor.MsgAcctHash).Data.(*ethcmn.Hash)
		store.state = s3StateWaitingParentInfo
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
	}
	return nil
}

func (store *StateSyncStore) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		s3StateUninit:            {actor.MsgParentInfo},
		s3StateWaitingUpdates:    {actor.CombinedName(actor.MsgUrlUpdate, actor.MsgAcctHash)},
		s3StateWaitingParentInfo: {actor.MsgParentInfo},
	}
}

func (store *StateSyncStore) GetCurrentState() int {
	return store.state
}

func (store *StateSyncStore) setSyncStatus(status *cmntyp.SyncStatus) error {
	store.status = status
	return store.sliceDB.Set("syncstatus", store.encode(status))
}

func (store *StateSyncStore) SetSyncStatus(ctx context.Context, status *cmntyp.SyncStatus, _ *int) error {
	return store.setSyncStatus(status)
}

func (store *StateSyncStore) getSyncStatus() *cmntyp.SyncStatus {
	if store.status == nil {
		store.status = &cmntyp.SyncStatus{}
		bs, err := store.sliceDB.Get("syncstatus")
		if err == nil {
			store.decode(bs, store.status)
		}
	}
	return store.status
}

func (store *StateSyncStore) GetSyncStatus(ctx context.Context, _ *int, status *cmntyp.SyncStatus) error {
	*status = *store.getSyncStatus()
	return nil
}

func (store *StateSyncStore) setSyncPoint(sp *cmntyp.SyncPoint) error {
	return store.sliceDB.Set("syncpoint", store.encode(sp))
}

func (store *StateSyncStore) SetSyncPoint(ctx context.Context, sp *cmntyp.SyncPoint, _ *int) error {
	store.sp = sp
	return store.setSyncPoint(sp)
}

func (store *StateSyncStore) getSyncPoint() *cmntyp.SyncPoint {
	if store.sp == nil {
		store.sp = &cmntyp.SyncPoint{}
		bs, err := store.sliceDB.Get("syncpoint")
		if err == nil {
			store.decode(bs, store.sp)
		}
	}
	return store.sp
}

func (store *StateSyncStore) GetSyncPoint(ctx context.Context, height *uint64, sp *cmntyp.SyncPoint) error {
	*sp = *store.getSyncPoint()
	if sp.To != *height {
		return errors.New("syncpoint not found")
	}
	return nil
}

func (store *StateSyncStore) InitSyncPoint(ctx context.Context, to *uint64, sp *cmntyp.SyncPoint) error {
	count := 0
	for i := 0; i < cmntyp.SlicePerSyncPoint; i++ {
		if response, err := store.readSliceFromKvDB(&cmntyp.SyncDataRequest{
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
	var parent cmntyp.ParentInfo
	intf.Router.Call("statestore", "GetParentInfo", &na, parent)
	// TODO: Set slice hashes.
	if err := store.setSyncPoint(&cmntyp.SyncPoint{
		From:   0,
		To:     *to,
		Slices: make([]ethcmn.Hash, cmntyp.SlicePerSyncPoint),
		Parent: &parent,
	}); err != nil {
		return err
	}
	return nil
}

func (store *StateSyncStore) WriteSlice(ctx context.Context, slice *cmntyp.SyncDataResponse, _ *int) error {
	return store.sliceDB.Set(store.sliceKey(&slice.SyncDataRequest), store.encode(slice))
}

func (store *StateSyncStore) deleteSlice(slice *cmntyp.SyncDataRequest) error {
	return store.sliceDB.Delete(store.sliceKey(slice))
}

func (store *StateSyncStore) readSliceFromSyncPointDB(request *cmntyp.SyncDataRequest) (*cmntyp.SyncDataResponse, error) {
	// TODO: check request.To & request.From
	keys, values, err := store.spDB.Query(fmt.Sprintf("%s%02x", urlRoot, []byte{byte(request.Slice)}), nil)
	if err != nil {
		return nil, err
	}

	// TODO: Calculate slice hash.
	response := &cmntyp.SyncDataResponse{
		SyncDataRequest: *request,
		Data: store.encode(&storage.UrlUpdate{
			Keys:          keys,
			EncodedValues: values,
		}),
	}
	fmt.Printf("StateSyncStore.ReadSlice, load %d keys\n", len(keys))
	return response, nil
}

func (store *StateSyncStore) readSliceFromKvDB(request *cmntyp.SyncDataRequest) (*cmntyp.SyncDataResponse, error) {
	bs, err := store.sliceDB.Get(store.sliceKey(request))
	if err != nil {
		return nil, err
	}

	var response cmntyp.SyncDataResponse
	store.decode(bs, &response)
	if response.To != request.To {
		return nil, errors.New("slice not found")
	}
	return &response, nil
}

func (store *StateSyncStore) ReadSlice(ctx context.Context, request *cmntyp.SyncDataRequest, response *cmntyp.SyncDataResponse) error {
	var resp *cmntyp.SyncDataResponse
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

func (store *StateSyncStore) sliceKey(slice *cmntyp.SyncDataRequest) string {
	return fmt.Sprintf("%016x-%04x", slice.From, slice.Slice)
}

func (store *StateSyncStore) makeSyncPoint(from, to uint64) {
	status := *store.getSyncStatus()
	// Disable sync point.
	status.SyncPoint = 0
	store.setSyncStatus(&status)

	var parent *cmntyp.ParentInfo
	for i := from; i < to; i++ {
		var response cmntyp.SyncDataResponse
		err := store.ReadSlice(context.Background(), &cmntyp.SyncDataRequest{
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

		err = store.deleteSlice(&cmntyp.SyncDataRequest{
			From:  i,
			To:    i + 1,
			Slice: 0,
		})
		if err != nil {
			fmt.Printf("[StateSyncStore] deleteSlice(%d) failed, err = %v\n", i, err)
		}
	}

	// TODO: Set slice hashes.
	store.setSyncPoint(&cmntyp.SyncPoint{
		From:   0,
		To:     to,
		Slices: make([]ethcmn.Hash, cmntyp.SlicePerSyncPoint),
		Parent: parent,
	})

	// Enable sync point.
	status.SyncPoint = to
	store.setSyncStatus(&status)
}
