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
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
)

type BlockStore struct {
	db *mstypes.BlockCaches
}

func NewBlockStore() actor.Business {
	return &BlockStore{}
}

func (bs *BlockStore) Config(params map[string]interface{}) {
	bs.db = mstypes.NewBlockCaches(params["storage_block_path"].(string), params["cache_block_size"].(int))
}

func (bs *BlockStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (bs *BlockStore) Outputs() map[string]int {
	return map[string]int{}
}

func (bs *BlockStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Save", bs.Save)
	// reg.Register("SavePendingBlock", bs.SavePendingBlock)
	reg.Register("GetByHeight", bs.GetByHeight)
	reg.Register("GetTransaction", bs.GetTransaction)
}

func (bs *BlockStore) RpcConfig() (string, int) {
	return "blockstore", 20
}

func (bs *BlockStore) Save(ctx *actor.ActionContext) error {
	block := ctx.RPC.Request.(*mtypes.MonacoBlock)
	bs.db.Save(block.Height, block)
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

// func (bs *BlockStore) SavePendingBlock(ctx *actor.ActionContext) error {
// 	block := ctx.RPC.Request.(*mtypes.MonacoBlock)
// 	bs.db.CacheOnly(block.Height, block)
// 	ctx.ExecCtx.SendRpcResponse("", nil)
// 	return nil
// }

func (bs *BlockStore) GetByHeight(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(uint64)
	ctx.ExecCtx.SendRpcResponse("", bs.db.Query(height))
	return nil
}

func (bs *BlockStore) GetTransaction(ctx *actor.ActionContext) error {
	position := ctx.RPC.Request.(*mstypes.Position)
	ctx.ExecCtx.SendRpcResponse("", bs.db.QueryTx(position.Height, position.IdxInBlock))
	return nil
}
