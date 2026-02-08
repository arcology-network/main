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
	"github.com/arcology-network/common-lib/storage/filedb"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	"github.com/arcology-network/streamer/actor"
)

type SaveIndexRequest struct {
	Height uint64
	Keys   []string
	Hash   string
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

func NewIndexerStore() actor.Business {
	return &IndexerStore{}
}

func (is *IndexerStore) Config(params map[string]interface{}) {
	filedb, err := filedb.NewFileDB(params["storage_index_path"].(string), uint32(params["storage_index_shards"].(int)), uint8(params["storage_index_depts"].(int)))
	if err != nil {
		panic("create filedb err!:" + err.Error())
	}
	is.db = mstypes.NewIndexer(filedb, params["cache_index_size"].(int))
}

func (is *IndexerStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (is *IndexerStore) Outputs() map[string]int {
	return map[string]int{}
}

func (is *IndexerStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Save", is.Save)
	// reg.Register("SaveBlockHash", is.SaveBlockHash)
	reg.Register("GetHeightByHash", is.GetHeightByHash)
	reg.Register("GetPosition", is.GetPosition)
	reg.Register("GetBlockHashes", is.GetBlockHashes)
}

func (is *IndexerStore) RpcConfig() (string, int) {
	return "indexerstore", 20
}

func (is *IndexerStore) Save(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*SaveIndexRequest)
	is.db.Add(request.Height, request.Keys, request.IsSave)
	is.db.AddBlockHashHeight(request.Height, request.Hash, request.IsSave)
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

// func (is *IndexerStore) SaveBlockHash(ctx *actor.ActionContext) error {
// 	request := ctx.RPC.Request.(*SaveIndexBlockHashRequest)
// 	is.db.AddBlockHashHeight(request.Height, request.Hash, request.IsSave)
// 	ctx.ExecCtx.SendRpcResponse("", nil)
// 	return nil
// }

func (is *IndexerStore) GetHeightByHash(ctx *actor.ActionContext) error {
	hash := ctx.RPC.Request.(string)
	ctx.ExecCtx.SendRpcResponse("", is.db.QueryBlockHashHeight(hash))
	return nil
}

func (is *IndexerStore) GetPosition(ctx *actor.ActionContext) error {
	hash := ctx.RPC.Request.(string)
	ctx.ExecCtx.SendRpcResponse("", is.db.QueryPosition(hash))
	return nil
}

func (is *IndexerStore) GetBlockHashes(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(uint64)
	ctx.ExecCtx.SendRpcResponse("", is.db.GetBlockHashesByHeightFromCache(height))
	return nil
}
