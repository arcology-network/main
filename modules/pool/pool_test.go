//go:build !CI

package pool

import (
	"math/big"
	"testing"

	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/exp/mempool"
	badgerpk "github.com/arcology-network/common-lib/storage/badger"
	cmntyp "github.com/arcology-network/common-lib/types"
	apihandler "github.com/arcology-network/evm-adaptor/apihandler"
	adaptorcommon "github.com/arcology-network/evm-adaptor/eth"
	ccurl "github.com/arcology-network/storage-committer"
	ccurlcommon "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/storage-committer/commutative"
	"github.com/arcology-network/storage-committer/interfaces"
	stgproxy "github.com/arcology-network/storage-committer/storage/proxy"
	"github.com/arcology-network/storage-committer/storage/statestore"
	cache "github.com/arcology-network/storage-committer/storage/writecache"
	evmCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

func intDb() *statestore.StateStore {
	db := stgproxy.NewStoreProxy().EnableCache()
	return statestore.NewStateStore(db)
}

func TestPoolWithUncheckedTx(t *testing.T) {

	db := intDb()

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
	if p.TxByHash.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}
	if p.TxUnchecked.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}
}

func TestPoolWithCheckedTx(t *testing.T) {
	db := intDb()

	n := 500
	initAccounts(db.Store(), 0, n)
	txs := genCheckedTxs(0, n, 1, 100)

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

	increaseNonce(db.Store(), picked)
	p.Clean(1)
	if p.TxByHash.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}
}

func TestPoolCherryPick(t *testing.T) {

	db := intDb()

	n := 500
	initAccounts(db.Store(), 0, n*2)
	batch1 := genCheckedTxs(0, n, 1, 100)
	batch2 := genUncheckedTxs(n, n*2)

	p := NewPool(db, 100, false)
	p.Add(batch1, "tester", 1)

	clearList := make([]evmCommon.Hash, n*2)
	for i := range batch1 {
		clearList[i] = batch1[i].TxHash
	}
	for i := range batch2 {
		clearList[i+n] = batch2[i].TxHash
	}

	picked := p.CherryPick(clearList)
	if picked != nil {
		t.Fail()
	}

	picked = p.Add(batch2, "tester", 1)
	if len(picked) != len(clearList) {
		t.Fail()
	}

	p.Clean(1)
}

func TestPoolAdd(t *testing.T) {
	db := intDb()

	n := 500
	initAccounts(db.Store(), 0, n)

	p := NewPool(db, 100, false)
	txs := genCheckedTxs(0, n, 1, 100)
	p.Add(txs, "tester", 1)
	// Low nonce.
	txs = genCheckedTxs(0, n, 0, 100)
	p.Add(txs, "tester", 1)
	// High gas price.
	txs = genCheckedTxs(0, n, 1, 200)
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

	increaseNonce(db.Store(), picked)
	p.Clean(1)
	if p.TxByHash.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}
}

func TestPoolReapEmptyTxSender(t *testing.T) {
	db := intDb()

	n := 500
	initAccounts(db.Store(), 0, n)

	p := NewPool(db, 100, false)
	txs := genCheckedTxs(0, n, 1, 100)
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

	increaseNonce(db.Store(), picked)
	p.Clean(1)
	if p.TxByHash.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}

	reaped = p.Reap(n)
	if len(reaped) != n-100 {
		t.Fail()
	}
}

func TestPoolCleanObsolete(t *testing.T) {
	db := intDb()

	n := 500
	initAccounts(db.Store(), 0, n)

	p := NewPool(db, 100, false)
	txs := genCheckedTxs(0, n, 1, 100)
	p.Add(txs, "tester", 1)
	txs = genCheckedTxs(0, n/2, 3, 100)
	p.Add(txs, "tester", 1)

	reaped := p.Reap(n)
	if len(reaped) != n {
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

	increaseNonce(db.Store(), picked)
	p.Clean(1)
	if p.TxByHash.Size() != uint32(n/2) || p.TxBySender.Size() != uint32(n) {
		t.Fail()
	}

	p.Clean(101)
	if p.TxByHash.Size() != uint32(n/2) || p.TxBySender.Size() != uint32(n/2) {
		t.Fail()
	}

	p.Clean(201)
	if p.TxByHash.Size() != 0 || p.TxBySender.Size() != 0 {
		t.Fail()
	}
}

func initdb(path string) (interfaces.Datastore, *badgerpk.ParaBadgerDB) {
	badger := badgerpk.NewParaBadgerDB(path, common.Remainder)

	db := stgproxy.NewStoreProxy().EnableCache()

	db.Inject(ccurlcommon.ETH10_ACCOUNT_PREFIX, commutative.NewPath())
	return db, badger
}

func initAccounts(db interfaces.Datastore, from, to int) {
	api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
		return cache.NewWriteCache(db, 32, 1)
	}, func(cache *cache.WriteCache) { cache.Clear() }))

	stateDB := adaptorcommon.NewImplStateDB(api)
	stateCommitter := ccurl.NewStorageCommitter(db)
	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i := from; i < to; i++ {
		address := evmCommon.BytesToAddress([]byte{byte(i / 256), byte(i % 256)})
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, new(big.Int).SetUint64(100))
		stateDB.SetNonce(address, 0)
	}
	_, transitions := api.WriteCache().(*cache.WriteCache).ExportAll()
	stateCommitter.Import(transitions)
	stateCommitter.Precommit([]uint32{0})
	stateCommitter.Commit(0)
}

func increaseNonce(db interfaces.Datastore, txs []*cmntyp.StandardTransaction) {
	api := apihandler.NewAPIHandler(mempool.NewMempool[*cache.WriteCache](16, 1, func() *cache.WriteCache {
		return cache.NewWriteCache(db, 32, 1)
	}, func(cache *cache.WriteCache) { cache.Clear() }))

	stateDB := adaptorcommon.NewImplStateDB(api)
	stateCommitter := ccurl.NewStorageCommitter(db)
	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i := range txs {
		address := evmCommon.BytesToAddress(txs[i].NativeMessage.From.Bytes())
		stateDB.SetNonce(address, 0)
	}
	_, transitions := api.WriteCache().(*cache.WriteCache).ExportAll()
	stateCommitter.Import(transitions)
	stateCommitter.Precommit([]uint32{0})
	stateCommitter.Commit(0)
}

func genUncheckedTxs(from, to int) []*cmntyp.StandardTransaction {
	txs := make([]*cmntyp.StandardTransaction, to-from)
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
		txs[i-from] = &cmntyp.StandardTransaction{
			TxHash:        hash,
			NativeMessage: &msg,
		}
	}
	return txs
}

func genCheckedTxs(from, to int, nonce uint64, gasPrice uint64) []*cmntyp.StandardTransaction {
	txs := make([]*cmntyp.StandardTransaction, to-from)
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
		txs[i-from] = &cmntyp.StandardTransaction{
			TxHash:        hash,
			NativeMessage: &msg,
		}
	}
	return txs
}
