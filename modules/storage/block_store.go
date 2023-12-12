package storage

import (
	"context"

	types "github.com/arcology-network/common-lib/types"
	mstypes "github.com/arcology-network/main/modules/storage/types"
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

func (bs *BlockStore) Save(ctx context.Context, block *types.MonacoBlock, _ *int) error {
	bs.db.Save(block.Height, block)
	return nil
}

func (bs *BlockStore) SavePendingBlock(ctx context.Context, block *types.MonacoBlock, _ *int) error {
	bs.db.CacheOnly(block.Height, block)
	return nil
}

func (bs *BlockStore) GetByHeight(ctx context.Context, height *uint64, block **types.MonacoBlock) error {
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
