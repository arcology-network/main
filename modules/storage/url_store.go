package storage

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"math/big"

	"github.com/arcology-network/common-lib/cachedstorage"
	"github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/storage"
	"github.com/arcology-network/concurrenturl/interfaces"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	strtyp "github.com/arcology-network/main/modules/storage/types"
)

type UrlContainerGetRequest struct {
	Address       string
	Id            string
	ContainerType int
	Key           string
}

type UrlEthStorageGetRequest struct {
	Address string
	Key     string
}

const (
	ContainerTypeArray = iota
	ContainerTypeMap
	ContainerTypeQueue
)

type UrlStore struct {
	db      interfaces.Datastore
	indexer *MetaIndexer
}

func NewUrlStore() *UrlStore {
	return &UrlStore{
		indexer: NewMetaIndexer(),
	}
}

func (us *UrlStore) Init(ctx context.Context, db interfaces.Datastore, _ *int) error {
	us.db = db
	return nil
}

func (us *UrlStore) Query(ctx context.Context, pattern *string, response *storage.QueryResponse) error {
	keys, vals, err := us.db.Query(*pattern, cachedstorage.Under)
	if err != nil {
		return err
	}
	response.Keys = keys
	response.Values = vals
	return nil
}

func (us *UrlStore) Get(ctx context.Context, keys *[]string, values *[][]byte) error {
	objs := us.db.BatchRetrive(*keys, nil)
	datas := make([][]byte, len(objs))
	for i := range *keys {
		datas[i] = ccdb.Codec{}.Encode("", objs[i]) //urltyp.ToBytes(objs[i])
	}
	*values = datas
	return nil
}

func (us *UrlStore) GetNonce(ctx context.Context, address *string, nonce *uint64) error {
	non, err := strtyp.GetNonce(us.db, *address)
	if err != nil {
		return err
	}
	*nonce = uint64(non)
	return nil
}

func (us *UrlStore) GetBalance(ctx context.Context, address *string, balance **big.Int) error {
	var err error
	*balance, err = strtyp.GetBalance(us.db, *address)
	if err != nil {
		return err
	}
	return nil
}

func (us *UrlStore) GetCode(ctx context.Context, address *string, code *[]byte) error {
	var err error
	*code, err = strtyp.GetCode(us.db, *address)
	if err != nil {
		return err
	}
	return nil
}

func (us *UrlStore) GetEthStorage(ctx context.Context, request *UrlEthStorageGetRequest, value *[]byte) error {
	var err error
	*value, err = strtyp.GetStorage(us.db, request.Address, request.Key)
	if err != nil {
		return err
	}
	return nil
}

func (us *UrlStore) ApplyData(ctx context.Context, request *cmntyp.SyncDataRequest, _ *int) error {
	var numSlice int
	// var slices []ethcmn.Hash
	if request.To-request.From > 1 { // Sync point.
		numSlice = cmntyp.SlicePerSyncPoint
	} else { // Block.
		numSlice = 1
	}

	var parent *cmntyp.ParentInfo
	var schdState *SchdState
	for i := 0; i < numSlice; i++ {
		var response cmntyp.SyncDataResponse
		err := intf.Router.Call("statesyncstore", "ReadSlice", &cmntyp.SyncDataRequest{
			From:  request.From,
			To:    request.To,
			Slice: i,
		}, &response)
		if err != nil {
			return err
		}
		parent = response.Parent

		if response.SchdStates != nil {
			schdState = response.SchdStates.(*SchdState)
		}

		// TODO: data validation.
		// slices = append(slices, ethcmn.BytesToHash(response.Hash))

		var urlUpdate storage.UrlUpdate
		gob.NewDecoder(bytes.NewBuffer(response.Data)).Decode(&urlUpdate)
		common.ParallelExecute(
			func() {
				values := make([]interface{}, len(urlUpdate.EncodedValues))
				for i, v := range urlUpdate.EncodedValues {
					values[i] = ccdb.Codec{}.Decode(v, nil) //urltyp.FromBytes(v)
				}
				us.db.BatchInject(urlUpdate.Keys, values)
			},
			func() {
				us.indexer.Scan(urlUpdate.Keys, urlUpdate.EncodedValues)
			},
		)
	}

	var na int
	var status cmntyp.SyncStatus
	err := intf.Router.Call("statesyncstore", "GetSyncStatus", &na, &status)
	if err != nil {
		return err
	}

	if request.To-request.From > 1 {
		var sp cmntyp.SyncPoint
		err = intf.Router.Call("statesyncstore", "InitSyncPoint", &request.To, &sp)
		if err != nil {
			return err
		}
		status.SyncPoint = request.To

		var p cmntyp.ParentInfo
		intf.Router.Call("statestore", "GetParentInfo", &na, &p)
		parent = &p
	} else {
		var na int
		intf.Router.Call("schdstore", "DirectWrite", schdState, &na)
	}

	// Update state
	intf.Router.Call("statestore", "Save", &State{
		Height:     request.To,
		ParentHash: parent.ParentHash,
		ParentRoot: parent.ParentRoot,
	}, &na)

	status.Height = request.To
	return intf.Router.Call("statesyncstore", "SetSyncStatus", &status, &na)
}

func (us *UrlStore) RewriteMeta(ctx context.Context, _ *int, _ *int) error {
	keys, values := us.indexer.ExportMetas()
	fmt.Printf("[UrlStore.RewriteMeta] Update %d meta keys\n", len(keys))
	us.db.BatchInject(keys, values)
	return nil
}
