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

	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockStore struct {
	db *mstypes.BlockCaches
}

func NewBlockStore() *BlockStore {
	return &BlockStore{
		// TODO
		//db: NewBlockCaches("blockfiles", 100),
	}
}

func (bs *BlockStore) Config(params map[string]interface{}) {
	bs.db = mstypes.NewBlockCaches(params["storage_block_path"].(string), int(params["cache_block_size"].(float64)))
}

func (bs *BlockStore) Save(ctx context.Context, block *mtypes.MonacoBlock, _ *int) error {
	bs.db.Save(block.Height, block)
	return nil
}

func (bs *BlockStore) SavePendingBlock(ctx context.Context, block *mtypes.MonacoBlock, _ *int) error {
	bs.db.CacheOnly(block.Height, block)
	return nil
}

func (bs *BlockStore) GetByHeight(ctx context.Context, height *uint64, block **mtypes.MonacoBlock) error {
	*block = bs.db.Query(*height)
	// if b := bs.db.Query(*height); b != nil {
	// 	*block = *b
	// }
	return nil
}

func (bs *BlockStore) GetTransaction(ctx context.Context, position *mstypes.Position, tx **evmTypes.Transaction) error {
	*tx = bs.db.QueryTx(position.Height, position.IdxInBlock)
	return nil
}
