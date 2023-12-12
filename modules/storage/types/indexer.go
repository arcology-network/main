package types

import (
	"fmt"

	"github.com/arcology-network/common-lib/cachedstorage"
	"github.com/arcology-network/common-lib/codec"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type Position struct {
	Height     uint64
	IdxInBlock int
}

type SaveObject struct {
	keys  []string
	datas [][]byte
}

func (p *Position) Encode() []byte {
	buffers := [][]byte{
		codec.Uint64(p.Height).Encode(),
		codec.Uint32(p.IdxInBlock).Encode(),
	}
	return codec.Byteset(buffers).Encode()
}

func (p *Position) Decode(data []byte) *Position {
	buffers := [][]byte(codec.Byteset{}.Decode(data).(codec.Byteset))
	p.Height = uint64(codec.Uint64(0).Decode(buffers[0]).(codec.Uint64))
	p.IdxInBlock = int(codec.Uint32(0).Decode(buffers[1]).(codec.Uint32))
	return p
}

type Indexer struct {
	Caches       *DataCache
	CachesHeight *DataCache
	Db           *cachedstorage.FileDB

	objChan  chan *SaveObject
	exitChan chan bool
}

func NewIndexer(filedb *cachedstorage.FileDB, cacheSize int) *Indexer {
	indexer := Indexer{
		Caches:       NewDataCache(cacheSize),
		CachesHeight: NewDataCache(cacheSize),
		Db:           filedb,

		objChan:  make(chan *SaveObject, 50),
		exitChan: make(chan bool),
	}

	return &indexer
}

func (indexer *Indexer) QueryBlockHashHeight(hash string) uint64 {
	height := indexer.CachesHeight.Query(hash)
	if height != nil {
		return height.(uint64)
	}
	data, err := indexer.Db.Get(hash)
	if err != nil {
		return 0
	}
	hashHeight := uint64(codec.Uint64(0).Decode(data).(codec.Uint64))
	indexer.AddBlockHashHeight(hashHeight, hash, false)
	return hashHeight
}
func (indexer *Indexer) AddBlockHashHeight(height uint64, hash string, isSave bool) {

	indexer.CachesHeight.Add(height, []string{hash}, []interface{}{height})
	if isSave {
		indexer.Db.Set(indexer.GetHashHeightKey(hash), codec.Uint64(height).Encode())
	}

}

func (indexer *Indexer) QueryPosition(hash string) *Position {
	position := indexer.Caches.Query(hash)
	if position != nil {
		return position.(*Position)
	}

	data, err := indexer.Db.Get(hash)
	if err != nil || len(data) == 0 {
		return nil
	}
	p := &Position{}
	p.Decode(data)

	keys := indexer.GetBlockHashesByHeightFromDb(p.Height)
	indexer.Add(p.Height, keys, false)

	return p
}

func (indexer *Indexer) Add(height uint64, keys []string, isSave bool) {
	hashs := make([]byte, len(keys)*evmCommon.HashLength)
	positions := make([]interface{}, len(keys))
	datas := make([][]byte, len(keys))
	for i, _ := range keys {
		p := Position{
			Height:     height,
			IdxInBlock: i,
		}
		positions[i] = &p
		if isSave {
			datas[i] = p.Encode()
		}

		copy(hashs[i*evmCommon.HashLength:], []byte(keys[i]))
	}
	indexer.Caches.Add(height, keys, positions)

	if isSave {
		indexer.Db.BatchSet(keys, datas)
		indexer.Db.Set(indexer.GetHashesInBlockKey(height), hashs)
	}
}

func (indexer *Indexer) GetHashesInBlockKey(height uint64) string {
	return fmt.Sprintf("hashesInBlock-%v", height)
}
func (indexer *Indexer) GetHashHeightKey(hash string) string {
	return fmt.Sprintf("hashHeight-%v", hash)
}

func (indexer *Indexer) GetBlockHashesByHeightFromDb(height uint64) []string {
	datas, err := indexer.Db.Get(indexer.GetHashesInBlockKey(height))
	if err != nil {
		return []string{}
	}

	counter := len(datas) / evmCommon.HashLength
	hashes := make([]string, counter)
	for i := range hashes {
		data := datas[i*evmCommon.HashLength : (i+1)*evmCommon.HashLength]
		hashes[i] = string(data)
	}
	return hashes
}

func (indexer *Indexer) GetBlockHashesByHeightFromCache(height uint64) []string {
	datas := indexer.Caches.GetHashes(height)
	if len(datas) == 0 {
		datas = indexer.GetBlockHashesByHeightFromDb(height)
		indexer.Add(height, datas, false)
	}
	if len(datas) == 0 {
		return []string{}
	}

	hashes := make([]string, len(datas))
	for i := range hashes {
		hashes[i] = fmt.Sprintf("%x", []byte(datas[i]))
	}
	return hashes
}
