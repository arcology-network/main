package types

import (
	"math/big"
	"os"
	"reflect"
	"testing"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethRlp "github.com/HPISTechnologies/3rd-party/eth/rlp"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/evm/params"
)

func newBlock(height uint64, idx int) (*types.MonacoBlock, []ethCommon.Hash) {
	header := &ethTypes.Header{
		ParentHash:  ethCommon.BytesToHash([]byte{byte(1 + idx), byte(2 + idx), byte(3 + idx), byte(4 + idx), 5, 6, 7, 8, 9, 10}),
		Number:      big.NewInt(common.Uint64ToInt64(height)),
		GasLimit:    uint64(params.GasLimit),
		Time:        big.NewInt(int64(idx)),
		Difficulty:  big.NewInt(1),
		Coinbase:    ethCommon.BytesToAddress([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}),
		Root:        ethCommon.BytesToHash([]byte{byte(1 + idx), byte(2 + idx), byte(3 + idx), 4, 5, 6, 7, 8, 9, 10}),
		GasUsed:     100,
		TxHash:      ethCommon.BytesToHash([]byte{byte(1 + idx), byte(2 + idx), 3, 4, 5, 6, 7, 8, 9, 10}),
		ReceiptHash: ethCommon.BytesToHash([]byte{byte(1 + idx), 2, 3, 4, 5, 6, 7, 8, 9, 10}),
	}
	ethHeader, err := ethRlp.EncodeToBytes(&header)
	if err != nil {
		return nil, nil
	}

	headers := [][]byte{}
	ethHeaders := make([]byte, len(ethHeader)+1)
	bz := 0
	bz += copy(ethHeaders[bz:], []byte{types.AppType_Eth})
	bz += copy(ethHeaders[bz:], ethHeader)

	headers = append(headers, ethHeaders)

	txSelected := make([][]byte, 2)
	txhashes := make([]ethCommon.Hash, 2)
	for i := range txSelected {
		txSelected[i] = ethCommon.Hex2Bytes("01" + txs[idx+i])
		txhashes[i] = ethCommon.HexToHash(hashes[idx+i])
	}

	block := &types.MonacoBlock{
		Height:  height,
		Headers: headers,
		Txs:     txSelected,
	}
	return block, txhashes
}

var (
	txs = []string{
		"f8a58001830f424094b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f80b844561291340000000000000000000000000000000000000000000000000000000000000000000000000000000000000000ab01a3bfc5de6b5fc481e18f274adbdba9b111f026a08005e92b48684d992b2d50f53c7f4a6a1796e834a7c21867deec24b323e8e410a0426ef084a18e22bec96b5dec726e1edf7ffaf029b9365343291f45df70e19c6d",
		"f8a50101830f424094b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f80b84456129134000000000000000000000000000000000000000000000000000000000000000100000000000000000000000021522c86a586e696961b68aa39632948d9f1117026a04738397c49cab9a29f0b39a4b1a55d413041cb85bbd31d2601830511c785e97ca043b527161e201f6e38ad3778246667dd0977695cf4eaacef35b71531763a6226",
		"f8a50201830f424094b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f80b844561291340000000000000000000000000000000000000000000000000000000000000002000000000000000000000000a75cd05bf16bbea1759de2a66c0472131bc5bd8d26a010ba6efa1848798ad6154e0630e600bad511ffcdc8e19daaf91efd3e6a0a7052a06e2fd129bbd687c26188629ceb028bfe2a252ecb44bc498bda90fe56c2d4162a",
		"f8a50301830f424094b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f80b8445612913400000000000000000000000000000000000000000000000000000000000000030000000000000000000000002c7161284197e40e83b1b657e98b3bb8ff3c90ed26a01a9ce72ef273960541312c79fe59b3803dc28273a3b23a8610d449ff2808efe2a02ad879921402b08578743f3fc24cad6e9bdd5ccf6db14b3ae66137bcba523d57",
		"f8a50401830f424094b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f80b84456129134000000000000000000000000000000000000000000000000000000000000000400000000000000000000000057170608ae58b7d62dcdc3cbdb564c05ddbb7eee25a0d2179b8f517d29791ac37ff9d5ae9929b35fc71c93b10954975a59d99b7b961fa0739fa47eb5c1ed24764dd177007bdb5ec511dd258cc13a6b95d2c47df959c182",
		"f8a50501830f424094b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f80b8445612913400000000000000000000000000000000000000000000000000000000000000050000000000000000000000009f79316c20f3f83fcf43dee8a1cea185a47a5c4525a08e37aa8c5b8b361db356143e47b66898c021e2c442e81f27c0f1236a12b23b92a07603c2da5e2cf31dcb476698e70f63786ff6ada044538d92d18e8b610e692f14",
	}
	hashes = []string{
		"6246cce0e391ac17ed6718e0d13165e48591cbb0902adeebee2f73936c443435",
		"4f3ca3deb97e66bcb3c846e678f58836266c7efbdf17fbfaef2cfadb2342f306",
		"0d831a47510c0793ff3d9441829cf7f413d61994055b995e7bfbacff56a7a047",
		"820a6ca96d8fab2d5ee3d768c98768270008fa53bbb428b707b4fca0030b3873",
		"440f479d950c3b6ce7e05d733fe4deda522f00510472545c4f94613dcc1399c8",
		"2361cde0c325b3b18bb2746339d82ccdb21d558eb1af080b959de2a9adaeede3",
	}
)

func TestBlockCache(t *testing.T) {
	cache := NewBlockCaches("blockfiles", 2)
	block1, _ := newBlock(1, 0)
	cache.Save(block1.Height, block1)

	queryResult := cache.Query(1)
	if !reflect.DeepEqual(*queryResult, *block1) {
		t.Error("cache save get Error")
		return
	}

	block2, _ := newBlock(2, 2)
	cache.Save(block2.Height, block2)

	queryResult = cache.Query(2)
	if !reflect.DeepEqual(*queryResult, *block2) {
		t.Error("cache save get Error")
		return
	}

	block3, _ := newBlock(3, 4)
	cache.Save(block3.Height, block3)
	heights := []uint64{2, 3}
	if !reflect.DeepEqual(heights, cache.caches.DataHeights) {
		t.Error("cache remove Error")
		return
	}
	queryResult = cache.Query(1)
	if !reflect.DeepEqual(*queryResult, *block1) {
		t.Error("cache save get Error")
		return
	}
	heights = []uint64{3, 1}
	if !reflect.DeepEqual(heights, cache.caches.DataHeights) {
		t.Error("cache remove Error")
		return
	}

	tx := cache.QueryTx(3, 1)
	data := ethCommon.Hex2Bytes("01" + txs[5])
	otx := new(ethTypes.Transaction)
	if err := ethRlp.DecodeBytes(data[1:], otx); err != nil {
		t.Error("tx decode Error")
	}
	if !reflect.DeepEqual(*tx, *otx) {
		t.Error("cache save get Error")
		return
	}

	os.RemoveAll("blockfiles")
}
