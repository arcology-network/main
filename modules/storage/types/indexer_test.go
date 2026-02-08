package types

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/arcology-network/common-lib/storage/filedb"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

func TestSave(t *testing.T) {
	txhashes := []evmCommon.Hash{
		evmCommon.BytesToHash([]byte{1, 2, 3, 4, 5, 6, 7, 8}),
		evmCommon.BytesToHash([]byte{11, 12, 13, 14, 51, 16, 17, 18}),
	}
	keys := make([]string, len(txhashes))
	for i := range keys {
		keys[i] = string(txhashes[i].Bytes())
	}
	blockHash := evmCommon.BytesToHash([]byte{101, 102, 103, 104, 105, 106, 107, 108})

	filedb, err := filedb.NewFileDB("./indexer", uint32(128), uint8(2))
	if err != nil {
		panic("create filedb err!:" + err.Error())
	}
	db := NewIndexer(filedb, 100)

	db.Add(10, keys, true)

	hashstr := string(blockHash.Bytes())
	db.AddBlockHashHeight(10, hashstr, true)

	reth := db.QueryBlockHashHeight(hashstr)
	if reth.Cmp(big.NewInt(10)) != 0 {
		t.Fatal("Save and query err")
	}

}
func TestByte(t *testing.T) {
	txs := [][]byte{
		evmCommon.Hex2Bytes("0002f8667601010382c24294b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f8084a523b88ac080a025665926933f352bcac4d9665cf78b930a9fa6089939b64385ab463b9a8b84d8a01999d62d473e0baf86c48c92f9b1f9462ea0882d870dff879089ca177fc681bc"),
		evmCommon.Hex2Bytes("0002f8667601010382c24294b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f8084a523b88ac080a0b72738d59e994e96aa51d20440b856aaf491c9020c8f884b1692c309d02a76bea03d93532ddab895d464c9640988e5dec07bd770e31a6cb4c6b5d2e2f4125040c6"),
	}
	fmt.Printf("aa:%v\n", txs)
}
