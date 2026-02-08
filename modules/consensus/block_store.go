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

package consensus

import (
	"github.com/arcology-network/consensus-engine/state"
	contyp "github.com/arcology-network/consensus-engine/types"
	"github.com/arcology-network/streamer/actor"
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
	sender  actor.OutboundSender
	from    string
}

func newBlockStore(service string, sender actor.OutboundSender) state.BlockStore {
	return &blockStore{
		service: service,
		sender:  sender,
		from:    "consensus",
	}
}

func (bs *blockStore) Base() int64 {
	base, err := bs.sender.SendSync(bs.service, "Base", "", 0, bs.from)
	if err != nil {
		return 0
	}
	return base.(int64)
}

func (bs *blockStore) Height() int64 {
	height, err := bs.sender.SendSync(bs.service, "Height", "", 0, bs.from)
	if err != nil {
		return 0
	}
	return height.(int64)
}

func (bs *blockStore) Size() int64 {
	size, err := bs.sender.SendSync(bs.service, "Size", "", 0, bs.from)
	if err != nil {
		return 0
	}
	return size.(int64)
}

func (bs *blockStore) LoadBaseMeta() *contyp.BlockMeta {
	bm, err := bs.sender.SendSync(bs.service, "LoadBaseMeta", "", 0, bs.from)
	if err != nil {
		return nil
	}
	return bm.(*contyp.BlockMeta)
}

func (bs *blockStore) LoadBlockMeta(height int64) *contyp.BlockMeta {
	bm, err := bs.sender.SendSync(bs.service, "LoadBlockMeta", height, 0, bs.from)
	if err != nil {
		return nil
	}
	return bm.(*contyp.BlockMeta)
}

func (bs *blockStore) LoadBlock(height int64) *contyp.Block {
	b, err := bs.sender.SendSync(bs.service, "LoadBlock", height, 0, bs.from)
	if err != nil {
		return nil
	}
	return b.(*contyp.Block)
}

func (bs *blockStore) SaveBlock(block *contyp.Block, blockParts *contyp.PartSet, seenCommit *contyp.Commit) {
	bs.sender.SendSync(
		bs.service,
		"SaveBlock",
		&SaveBlockRequest{
			Block:      block,
			BlockParts: blockParts,
			SeenCommit: seenCommit,
		}, 0, bs.from)

}

func (bs *blockStore) SaveBlockAsync(block *contyp.Block, blockParts *contyp.PartSet, seenCommit *contyp.Commit) {
	bs.sender.SendSync(
		bs.service,
		"SaveBlockAsync",
		&SaveBlockRequest{
			Block:      block,
			BlockParts: blockParts,
			SeenCommit: seenCommit,
		}, 0, bs.from)
}

func (bs *blockStore) PruneBlocks(height int64) (uint64, error) {
	pruned, err := bs.sender.SendSync(bs.service, "PruneBlocks", height, 0, bs.from)
	if err != nil {
		return 0, err
	}
	return pruned.(uint64), err
}

func (bs *blockStore) LoadBlockByHash(hash []byte) *contyp.Block {
	b, err := bs.sender.SendSync(bs.service, "LoadBlockByHash", hash, 0, bs.from)
	if err != nil {
		return nil
	}
	return b.(*contyp.Block)
}

func (bs *blockStore) LoadBlockPart(height int64, index int) *contyp.Part {
	p, err := bs.sender.SendSync(bs.service, "LoadBlockPart", &LoadBlockPartRequest{
		Height: height,
		Index:  index,
	}, 0, bs.from)
	if err != nil {
		return nil
	}

	return p.(*contyp.Part)
}

func (bs *blockStore) LoadBlockCommit(height int64) *contyp.Commit {
	c, err := bs.sender.SendSync(bs.service, "LoadBlockCommit", height, 0, bs.from)
	if err != nil {
		return nil
	}
	return c.(*contyp.Commit)
}

func (bs *blockStore) LoadSeenCommit(height int64) *contyp.Commit {
	c, err := bs.sender.SendSync(bs.service, "LoadSeenCommit", height, 0, bs.from)
	if err != nil {
		return nil
	}
	return c.(*contyp.Commit)
}

func (bs *blockStore) SaveSeenCommit(height int64, seenCommit *contyp.Commit) error {
	_, err := bs.sender.SendSync(bs.service,
		"SaveSeenCommit",
		&SaveSeenCommitRequest{
			Height:     height,
			SeenCommit: seenCommit,
		}, 0, bs.from)
	return err
}
