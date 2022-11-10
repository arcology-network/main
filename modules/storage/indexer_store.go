package storage

import (
	"context"

	"github.com/arcology-network/common-lib/cachedstorage"
	mstypes "github.com/arcology-network/main/modules/storage/types"
)

type SaveIndexRequest struct {
	Height uint64
	keys   []string
	IsSave bool
}

type SaveIndexBlockHashRequest struct {
	Height uint64
	Hash   string
	IsSave bool
}

type IndexerStore struct {
	db *mstypes.Indexer
}

func NewIndexerStore() *IndexerStore {
	return &IndexerStore{
		// TODO
		//db: NewIndexer(filedb, 100),
	}
}

func (is *IndexerStore) Config(params map[string]interface{}) {
	filedb, err := cachedstorage.NewFileDB(params["storage_index_path"].(string), uint32(params["storage_index_shards"].(float64)), uint8(params["storage_index_depts"].(float64)))
	if err != nil {
		panic("create filedb err!:" + err.Error())
	}
	is.db = mstypes.NewIndexer(filedb, int(params["cache_index_size"].(float64)))
}

func (is *IndexerStore) Save(ctx context.Context, request *SaveIndexRequest, _ *int) error {
	is.db.Add(request.Height, request.keys, request.IsSave)
	return nil
}
func (is *IndexerStore) SaveBlockHash(ctx context.Context, request *SaveIndexBlockHashRequest, _ *int) error {
	is.db.AddBlockHashHeight(request.Height, request.Hash, request.IsSave)
	return nil
}

func (is *IndexerStore) GetHeightByHash(ctx context.Context, hash *string, height *uint64) error {
	h := is.db.QueryBlockHashHeight(*hash)
	*height = h
	return nil
}

func (is *IndexerStore) GetPosition(ctx context.Context, hash *string, position **mstypes.Position) error {
	*position = is.db.QueryPosition(*hash)
	return nil
}

func (is *IndexerStore) GetBlockHashes(ctx context.Context, height *uint64, hashes *[]string) error {
	*hashes = is.db.GetBlockHashesByHeightFromCache(*height)
	return nil
}
