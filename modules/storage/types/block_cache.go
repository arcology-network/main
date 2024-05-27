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

package types

import (
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

type BlockCaches struct {
	caches *DataCache
	db     *RawFile
}

func NewBlockCaches(path string, cache int) *BlockCaches {
	return &BlockCaches{
		caches: NewDataCache(cache),
		db:     NewRawFiles(path),
	}
}
func (rc *BlockCaches) QueryTx(height uint64, idx int) *evmTypes.Transaction {
	block := rc.Query(height)
	if block == nil || idx >= len(block.Txs) {
		return nil
	}
	// data := block.Txs[idx][1:]
	otx := new(evmTypes.Transaction)

	if err := otx.UnmarshalBinary(block.Txs[idx][1:]); err != nil {
		return nil
	}
	return otx
}
func (rc *BlockCaches) Query(height uint64) *mtypes.MonacoBlock {
	heightstr := fmt.Sprintf("%v", height)
	block := rc.caches.Query(heightstr)
	if block != nil {
		return block.(*mtypes.MonacoBlock)
	}

	data, err := rc.db.Read(rc.db.GetFilename(height))
	if err != nil {
		return nil
	}

	blockobj := mtypes.MonacoBlock{}
	err = blockobj.GobDecode(data)
	//err = common.GobDecode(data, &blockobj)
	if err != nil {
		return nil
	}
	rc.caches.Add(height, []string{heightstr}, []interface{}{&blockobj})
	return &blockobj
}

func (rc *BlockCaches) Save(height uint64, block *mtypes.MonacoBlock) {
	data, err := block.GobEncode()
	if err != nil {
		return
	}
	key := fmt.Sprintf("%v", height)
	rc.caches.Add(height, []string{key}, []interface{}{block})
	rc.db.Write(rc.db.GetFilename(height), data)
}

func (rc *BlockCaches) CacheOnly(height uint64, block *mtypes.MonacoBlock) {
	key := fmt.Sprintf("%v", height)
	rc.caches.Add(height, []string{key}, []interface{}{block})
}
