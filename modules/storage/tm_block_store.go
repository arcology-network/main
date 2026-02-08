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
	"github.com/arcology-network/consensus-engine/store"
	conwrk "github.com/arcology-network/main/modules/consensus"
	"github.com/arcology-network/streamer/actor"
	tmdb "github.com/tendermint/tm-db"
)

type TmBlockStore struct {
	impl *store.BlockStore
}

func NewTmBlockStore() actor.Business {
	return &TmBlockStore{}
}

func (bs *TmBlockStore) Config(params map[string]interface{}) {
	db, err := tmdb.NewDB(params["storage_tmblock_name"].(string), tmdb.GoLevelDBBackend, params["storage_tmblock_dir"].(string))
	if err != nil {
		panic(err)
	}
	bs.impl = store.NewBlockStore(db)
}

func (bs *TmBlockStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (bs *TmBlockStore) Outputs() map[string]int {
	return map[string]int{}
}

func (bs *TmBlockStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Base", bs.Base)
	reg.Register("Height", bs.Height)
	reg.Register("Size", bs.Size)
	reg.Register("LoadBaseMeta", bs.LoadBaseMeta)
	reg.Register("LoadBlockMeta", bs.LoadBlockMeta)
	reg.Register("LoadBlock", bs.LoadBlock)
	reg.Register("SaveBlock", bs.SaveBlock)
	reg.Register("SaveBlockAsync", bs.SaveBlockAsync)
	reg.Register("PruneBlocks", bs.PruneBlocks)
	reg.Register("LoadBlockByHash", bs.LoadBlockByHash)
	reg.Register("LoadBlockPart", bs.LoadBlockPart)
	reg.Register("LoadBlockCommit", bs.LoadBlockCommit)
	reg.Register("LoadSeenCommit", bs.LoadSeenCommit)
	reg.Register("SaveSeenCommit", bs.SaveSeenCommit)
}

func (bs *TmBlockStore) RpcConfig() (string, int) {
	return "tmblockstore", 20
}

func (bs *TmBlockStore) Base(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", bs.impl.Base())
	return nil
}

func (bs *TmBlockStore) Height(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", bs.impl.Height())
	return nil
}

func (bs *TmBlockStore) Size(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", bs.impl.Size())
	return nil
}

func (bs *TmBlockStore) LoadBaseMeta(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadBaseMeta())
	return nil
}

func (bs *TmBlockStore) LoadBlockMeta(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadBlockMeta(ctx.RPC.Request.(int64)))
	return nil
}

func (bs *TmBlockStore) LoadBlock(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadBlock(ctx.RPC.Request.(int64)))
	return nil
}

func (bs *TmBlockStore) SaveBlock(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*conwrk.SaveBlockRequest)
	bs.impl.SaveBlockEx(request.Block, request.BlockParts, request.SeenCommit)
	ctx.ExecCtx.SendRpcResponse("", "")
	return nil
}

func (bs *TmBlockStore) SaveBlockAsync(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*conwrk.SaveBlockRequest)
	bs.impl.SaveBlockAsync(request.Block, request.BlockParts, request.SeenCommit)
	ctx.ExecCtx.SendRpcResponse("", "")
	return nil
}

func (bs *TmBlockStore) PruneBlocks(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(int64)
	pruned, err := bs.impl.PruneBlocks(height)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", pruned)
	}
	return nil
}

func (bs *TmBlockStore) LoadBlockByHash(ctx *actor.ActionContext) error {
	hash := ctx.RPC.Request.([]byte)
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadBlockByHash(hash))
	return nil
}

func (bs *TmBlockStore) LoadBlockPart(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*conwrk.LoadBlockPartRequest)
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadBlockPart(request.Height, request.Index))
	return nil
}

func (bs *TmBlockStore) LoadBlockCommit(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(int64)
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadBlockCommit(height))
	return nil
}

func (bs *TmBlockStore) LoadSeenCommit(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(int64)
	ctx.ExecCtx.SendRpcResponse("", bs.impl.LoadSeenCommit(height))
	return nil
}

func (bs *TmBlockStore) SaveSeenCommit(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*conwrk.SaveSeenCommitRequest)
	err := bs.impl.SaveSeenCommit(request.Height, request.SeenCommit)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), "")
	} else {
		ctx.ExecCtx.SendRpcResponse("", "")
	}
	return nil
}
