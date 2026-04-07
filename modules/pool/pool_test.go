//go:build !CI

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

package pool

import (
	"math"
	"math/big"
	"testing"

	"github.com/arcology-network/common-lib/crdt/statecell"
	"github.com/arcology-network/common-lib/exp/mempool"
	cmntyp "github.com/arcology-network/common-lib/types"
	apihandler "github.com/arcology-network/eu/apihandler"
	statestore "github.com/arcology-network/state-engine"
	"github.com/arcology-network/state-engine/storage/proxy"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/holiman/uint256"

	ethimpl "github.com/arcology-network/eu/ethadaptor"
	statecache "github.com/arcology-network/state-engine/state/cache"
)

func intDb(filepath string) *statestore.StateStore {
	db := proxy.NewLevelDBStoreProxy(filepath, filepath, math.MaxUint64, &hashdb.Config{CleanCacheSize: 1024 * 1024 * 100})
	return statestore.NewStateStore(db)
}

func TestPoolWithUncheckedTx(t *testing.T) {

	db := intDb("TestPoolWithUncheckedTx")

	n := 500
	txs := genUncheckedTxs(0, n)

	p := NewPool(db, 100, false)
	p.Add(txs, "tester", 1)

	reaped := p.Reap(100)
	if len(reaped) != 100 {
		t.Fail()
	}

	clearList := make([]evmCommon.Hash, len(reaped))
	for i := range reaped {
		clearList[i] = reaped[i].TxHash
	}
	picked := p.CherryPick(clearList)
	if len(picked) != len(reaped) {
		t.Fail()
	}

	p.Clean(1)
	if p.TxByHash.Length() != uint64(n-len(reaped)) {
		t.Fail()
	}
	if p.TxUnchecked.Length() != uint64(n-len(reaped)) {
		t.Fail()
	}
}

func initAccounts(db *statestore.StateStore, from, to int) {
	ccRuntime := apihandler.NewConcurrentRuntime(
		0, // concurrent runtime ID
		mempool.NewMempool(
			16,
			1,
			func() *statecache.ExecutionStateCache {
				// When creating a new writecache, use store as the backend.
				return statecache.NewExecutionStateCache(db, 32, 1)
			},
			func(cache *statecache.ExecutionStateCache) {
				cache.Clear()
			}),
	)

	stateDB := ethimpl.NewImplStateDB(ccRuntime)

	// stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i := from; i < to; i++ {
		address := evmCommon.BytesToAddress([]byte{byte(i / 256), byte(i % 256)})
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, uint256.NewInt(100))
		stateDB.SetNonce(address, 0)
	}

	writeCache := db.ExecutionStateCache
	transitions := writeCache.Export(statecell.Sorter)

	db.Import(transitions)
	db.DebugPrecommit([]uint64{0})
	db.DebugCommit(0)
}

func increaseNonce(db *statestore.StateStore, txs []*cmntyp.StandardTransaction) {
	// api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
	// 	return cache.NewWriteCache(db, 32, 1)
	// }, func(cache *cache.WriteCache) { cache.Clear() }))
	// sstore := db //statestore.NewStateStore(db.(*stgproxy.StorageProxy))

	ccRuntime := apihandler.NewConcurrentRuntime(
		0, // concurrent runtime ID
		mempool.NewMempool(
			16,
			1,
			func() *statecache.ExecutionStateCache {
				// When creating a new writecache, use store as the backend.
				return statecache.NewExecutionStateCache(db, 32, 1)
			},
			func(cache *statecache.ExecutionStateCache) {
				cache.Clear()
			}),
	)

	stateDB := ethimpl.NewImplStateDB(ccRuntime)
	// stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i := range txs {
		address := evmCommon.BytesToAddress(txs[i].NativeMessage.From.Bytes())
		stateDB.SetNonce(address, 0)
	}
	writeCache := db.ExecutionStateCache
	transitions := writeCache.Export(statecell.Sorter)
	db.Import(transitions)
	db.DebugPrecommit([]uint64{0})
	db.DebugCommit(0)
}

func genUncheckedTxs(from, to int) []*cmntyp.StandardTransaction {
	rtxs := make([]*cmntyp.StandardTransaction, to-from)
	data := evmCommon.Hex2Bytes(txs[0])
	otx := new(evmTypes.Transaction)
	if err := otx.UnmarshalBinary(data); err != nil {
		return rtxs
	}
	for i := from; i < to; i++ {
		hash := evmCommon.BytesToHash([]byte{byte(i / 256), byte(i % 256)})
		msg := core.NewMessage(
			evmCommon.BytesToAddress([]byte{byte(i / 256), byte(i % 256)}),
			nil,
			0,
			nil,
			0,
			nil,
			nil,
			nil,
			false,
		)

		rtxs[i-from] = &cmntyp.StandardTransaction{
			TxHash:            hash,
			NativeMessage:     &msg,
			NativeTransaction: otx,
			TxRawData:         []byte{0, 2, 2},
			Source:            0,
			Signer:            1,
		}
	}
	return rtxs
}

func genCheckedTxs(from, to int, nonce uint64, gasPrice uint64) []*cmntyp.StandardTransaction {
	rtxs := make([]*cmntyp.StandardTransaction, to-from)
	data := evmCommon.Hex2Bytes(txs[0])
	otx := new(evmTypes.Transaction)
	if err := otx.UnmarshalBinary(data); err != nil {
		return rtxs
	}
	for i := from; i < to; i++ {
		hash := evmCommon.BytesToHash([]byte{byte(i / 256), byte(i % 256), byte(nonce), byte(gasPrice % 256)})
		msg := core.NewMessage(
			evmCommon.BytesToAddress([]byte{byte(i / 256), byte(i % 256)}),
			nil,
			nonce,
			nil,
			0,
			new(big.Int).SetUint64(gasPrice),
			nil,
			nil,
			true,
		)
		rtxs[i-from] = &cmntyp.StandardTransaction{
			TxHash:            hash,
			NativeMessage:     &msg,
			NativeTransaction: otx,
			TxRawData:         []byte{0, 2, 2},
			Source:            0,
			Signer:            1,
		}
	}
	return rtxs
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
