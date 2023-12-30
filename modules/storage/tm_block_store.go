package storage

import (
	"context"

	"github.com/arcology-network/consensus-engine/store"
	contyp "github.com/arcology-network/consensus-engine/types"
	conwrk "github.com/arcology-network/main/modules/consensus"
	tmdb "github.com/tendermint/tm-db"
)

type TmBlockStore struct {
	impl *store.BlockStore
}

func NewTmBlockStore() *TmBlockStore {
	// db, err := tmdb.NewDB("blockstore", tmdb.GoLevelDBBackend, "./")
	// if err != nil {
	// 	panic(err)
	// }

	return &TmBlockStore{
		// impl: store.NewBlockStore(db),
	}
}

func (bs *TmBlockStore) Config(params map[string]interface{}) {
	db, err := tmdb.NewDB(params["storage_tmblock_name"].(string), tmdb.GoLevelDBBackend, params["storage_tmblock_dir"].(string))
	if err != nil {
		panic(err)
	}

	bs.impl = store.NewBlockStore(db)
}

func (bs *TmBlockStore) Base(ctx context.Context, _ *int, ret *int64) error {
	*ret = bs.impl.Base()
	return nil
}

func (bs *TmBlockStore) Height(ctx context.Context, _ *int, ret *int64) error {
	*ret = bs.impl.Height()
	return nil
}

func (bs *TmBlockStore) Size(ctx context.Context, _ *int, ret *int64) error {
	*ret = bs.impl.Size()
	return nil
}

func (bs *TmBlockStore) LoadBaseMeta(ctx context.Context, _ *int, ret *contyp.BlockMeta) error {
	bm := bs.impl.LoadBaseMeta()
	if bm != nil {
		*ret = *bm
	}
	return nil
}

func (bs *TmBlockStore) LoadBlockMeta(ctx context.Context, height *int64, ret *contyp.BlockMeta) error {
	bm := bs.impl.LoadBlockMeta(*height)
	if bm != nil {
		*ret = *bm
	}
	return nil
}

func (bs *TmBlockStore) LoadBlock(ctx context.Context, height *int64, ret *contyp.Block) error {
	b := bs.impl.LoadBlock(*height)
	if b != nil {
		*ret = *b
	}
	return nil
}

func (bs *TmBlockStore) SaveBlock(ctx context.Context, request *conwrk.SaveBlockRequest, _ *int) error {
	bs.impl.SaveBlockEx(request.Block, request.BlockParts, request.SeenCommit)
	return nil
}

func (bs *TmBlockStore) SaveBlockAsync(ctx context.Context, request *conwrk.SaveBlockRequest, _ *int) error {
	bs.impl.SaveBlockAsync(request.Block, request.BlockParts, request.SeenCommit)
	return nil
}

func (bs *TmBlockStore) PruneBlocks(ctx context.Context, height *int64, ret *uint64) error {
	pruned, err := bs.impl.PruneBlocks(*height)
	*ret = pruned
	return err
}

func (bs *TmBlockStore) LoadBlockByHash(ctx context.Context, hash *[]byte, ret *contyp.Block) error {
	b := bs.impl.LoadBlockByHash(*hash)
	if b != nil {
		*ret = *b
	}
	return nil
}

func (bs *TmBlockStore) LoadBlockPart(ctx context.Context, request *conwrk.LoadBlockPartRequest, ret *contyp.Part) error {
	p := bs.impl.LoadBlockPart(request.Height, request.Index)
	if p != nil {
		*ret = *p
	}
	return nil
}

func (bs *TmBlockStore) LoadBlockCommit(ctx context.Context, height *int64, ret *contyp.Commit) error {
	c := bs.impl.LoadBlockCommit(*height)
	if c != nil {
		*ret = *c
	}
	return nil
}

func (bs *TmBlockStore) LoadSeenCommit(ctx context.Context, height *int64, ret *contyp.Commit) error {
	c := bs.impl.LoadSeenCommit(*height)
	if c != nil {
		*ret = *c
	}
	return nil
}

func (bs *TmBlockStore) SaveSeenCommit(ctx context.Context, request *conwrk.SaveSeenCommitRequest, _ *int) error {
	return bs.impl.SaveSeenCommit(request.Height, request.SeenCommit)
}
