package consensus

import (
	"github.com/arcology-network/consensus-engine/state"
	contyp "github.com/arcology-network/consensus-engine/types"
	intf "github.com/arcology-network/streamer/interface"
)

type SaveBlockRequest struct {
	Block      *contyp.Block
	BlockParts *contyp.PartSet
	SeenCommit *contyp.Commit
}

type LoadBlockPartRequest struct {
	Height int64
	Index  int
}

type SaveSeenCommitRequest struct {
	Height     int64
	SeenCommit *contyp.Commit
}

type blockStore struct {
	service string
}

func newBlockStore(service string) state.BlockStore {
	return &blockStore{service: service}
}

func (bs *blockStore) Base() int64 {
	var na int
	var base int64
	intf.Router.Call(bs.service, "Base", &na, &base)
	return base
}

func (bs *blockStore) Height() int64 {
	var na int
	var height int64
	intf.Router.Call(bs.service, "Height", &na, &height)
	return height
}

func (bs *blockStore) Size() int64 {
	var na int
	var size int64
	intf.Router.Call(bs.service, "Size", &na, &size)
	return size
}

func (bs *blockStore) LoadBaseMeta() *contyp.BlockMeta {
	var na int
	var bm contyp.BlockMeta
	intf.Router.Call(bs.service, "LoadBaseMeta", &na, &bm)
	return &bm
}

func (bs *blockStore) LoadBlockMeta(height int64) *contyp.BlockMeta {
	var bm contyp.BlockMeta
	intf.Router.Call(bs.service, "LoadBlockMeta", &height, &bm)
	return &bm
}

func (bs *blockStore) LoadBlock(height int64) *contyp.Block {
	var b contyp.Block
	intf.Router.Call(bs.service, "LoadBlock", &height, &b)
	return &b
}

func (bs *blockStore) SaveBlock(block *contyp.Block, blockParts *contyp.PartSet, seenCommit *contyp.Commit) {
	var na int
	intf.Router.Call(
		bs.service,
		"SaveBlock",
		&SaveBlockRequest{
			Block:      block,
			BlockParts: blockParts,
			SeenCommit: seenCommit,
		},
		&na,
	)
}

func (bs *blockStore) SaveBlockAsync(block *contyp.Block, blockParts *contyp.PartSet, seenCommit *contyp.Commit) {
	var na int
	intf.Router.Call(
		bs.service,
		"SaveBlockAsync",
		&SaveBlockRequest{
			Block:      block,
			BlockParts: blockParts,
			SeenCommit: seenCommit,
		},
		&na,
	)
}

func (bs *blockStore) PruneBlocks(height int64) (uint64, error) {
	var pruned uint64
	err := intf.Router.Call(bs.service, "PruneBlocks", &height, &pruned)
	return pruned, err
}

func (bs *blockStore) LoadBlockByHash(hash []byte) *contyp.Block {
	var b contyp.Block
	intf.Router.Call(bs.service, "LoadBlockByHash", &hash, &b)
	return &b
}

func (bs *blockStore) LoadBlockPart(height int64, index int) *contyp.Part {
	var p contyp.Part
	intf.Router.Call(
		bs.service,
		"LoadBlockPart",
		&LoadBlockPartRequest{
			Height: height,
			Index:  index,
		},
		&p,
	)
	return &p
}

func (bs *blockStore) LoadBlockCommit(height int64) *contyp.Commit {
	var c contyp.Commit
	intf.Router.Call(bs.service, "LoadBlockCommit", &height, &c)
	return &c
}

func (bs *blockStore) LoadSeenCommit(height int64) *contyp.Commit {
	var c contyp.Commit
	intf.Router.Call(bs.service, "LoadSeenCommit", &height, &c)
	return &c
}

func (bs *blockStore) SaveSeenCommit(height int64, seenCommit *contyp.Commit) error {
	var na int
	return intf.Router.Call(
		bs.service,
		"SaveSeenCommit",
		&SaveSeenCommitRequest{
			Height:     height,
			SeenCommit: seenCommit,
		},
		&na,
	)
}
