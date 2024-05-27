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
	"context"

	"github.com/arcology-network/common-lib/storage/filedb"
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
	filedb, err := filedb.NewFileDB(params["storage_index_path"].(string), uint32(params["storage_index_shards"].(float64)), uint8(params["storage_index_depts"].(float64)))
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
