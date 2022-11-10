package types

import (
	"fmt"

	ethRlp "github.com/arcology-network/3rd-party/eth/rlp"
	ethTypes "github.com/arcology-network/3rd-party/eth/types"
	"github.com/arcology-network/common-lib/types"
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
func (rc *BlockCaches) QueryTx(height uint64, idx int) *ethTypes.Transaction {
	block := rc.Query(height)
	if block == nil || idx >= len(block.Txs) {
		return nil
	}
	data := block.Txs[idx][1:]
	otx := new(ethTypes.Transaction)
	if err := ethRlp.DecodeBytes(data, otx); err != nil {
		return nil
	}
	return otx
}
func (rc *BlockCaches) Query(height uint64) *types.MonacoBlock {
	heightstr := fmt.Sprintf("%v", height)
	block := rc.caches.Query(heightstr)
	if block != nil {
		return block.(*types.MonacoBlock)
	}

	data, err := rc.db.Read(rc.db.GetFilename(height))
	if err != nil {
		return nil
	}

	blockobj := types.MonacoBlock{}
	err = blockobj.GobDecode(data)
	//err = common.GobDecode(data, &blockobj)
	if err != nil {
		return nil
	}
	rc.caches.Add(height, []string{heightstr}, []interface{}{&blockobj})
	return &blockobj
}

func (rc *BlockCaches) Save(height uint64, block *types.MonacoBlock) {
	data, err := block.GobEncode()
	if err != nil {
		return
	}
	key := fmt.Sprintf("%v", height)
	rc.caches.Add(height, []string{key}, []interface{}{block})
	rc.db.Write(rc.db.GetFilename(height), data)
}

func (rc *BlockCaches) CacheOnly(height uint64, block *types.MonacoBlock) {
	key := fmt.Sprintf("%v", height)
	rc.caches.Add(height, []string{key}, []interface{}{block})
}
