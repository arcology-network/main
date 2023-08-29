//go:build !CI

package pool

import (
	"math"
	"math/big"
	"testing"

	cstore "github.com/arcology-network/common-lib/cachedstorage"
	cmntyp "github.com/arcology-network/common-lib/types"
	ccurl "github.com/arcology-network/concurrenturl"
	ccurlcommon "github.com/arcology-network/concurrenturl/common"
	concurrenturlcommon "github.com/arcology-network/concurrenturl/common"
	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/interfaces"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/core"
	ccapi "github.com/arcology-network/vm-adaptor/api"
	"github.com/arcology-network/vm-adaptor/eth"
)

func TestPoolWithUncheckedTx(t *testing.T) {
	db, badger := initdb("./testdata/1/")
	defer badger.Close()

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
	db, badger := initdb("./testdata/2/")
	defer badger.Close()

	n := 500
	initAccounts(db, 0, n)
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

	increaseNonce(db, picked)
	p.Clean(1)
	if p.TxByHash.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}
}

func TestPoolCherryPick(t *testing.T) {
	db, badger := initdb("./testdata/3/")
	defer badger.Close()

	n := 500
	initAccounts(db, 0, n*2)
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
	db, badger := initdb("./testdata/4/")
	defer badger.Close()

	n := 500
	initAccounts(db, 0, n)

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

	increaseNonce(db, picked)
	p.Clean(1)
	if p.TxByHash.Size() != uint32(n-len(reaped)) {
		t.Fail()
	}
}

func TestPoolReapEmptyTxSender(t *testing.T) {
	db, badger := initdb("./testdata/5/")
	defer badger.Close()

	n := 500
	initAccounts(db, 0, n)

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

	increaseNonce(db, picked)
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
	db, badger := initdb("./testdata/6/")
	defer badger.Close()

	n := 500
	initAccounts(db, 0, n)

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

	increaseNonce(db, picked)
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

func initdb(path string) (interfaces.Datastore, *cstore.ParaBadgerDB) {
	badger := cstore.NewParaBadgerDB(path, ccurlcommon.Eth10AccountShard)
	db := cstore.NewDataStore(
		nil,
		cstore.NewCachePolicy(math.MaxUint64, 1),
		badger,
		// func(v interface{}) []byte { return urltyp.ToBytes(v) },
		// func(bytes []byte) interface{} { return urltyp.FromBytes(bytes) },
		func(v interface{}) []byte {
			return ccdb.Codec{}.Encode(v)
		},
		func(bytes []byte) interface{} {
			return ccdb.Codec{}.Decode(bytes)
		},
	)

	db.Inject(concurrenturlcommon.ETH10_ACCOUNT_PREFIX, commutative.NewPath())
	return db, badger
}

func initAccounts(db interfaces.Datastore, from, to int) {
	url := ccurl.NewConcurrentUrl(db)

	api := ccapi.NewAPI(url)
	stateDB := eth.NewImplStateDB(api)

	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i := from; i < to; i++ {
		address := evmCommon.BytesToAddress([]byte{byte(i / 256), byte(i % 256)})
		stateDB.CreateAccount(address)
		stateDB.SetBalance(address, new(big.Int).SetUint64(100))
		stateDB.SetNonce(address, 0)
	}
	_, transitions := url.ExportAll()
	url.Import(transitions)
	url.Sort()
	url.Finalize([]uint32{0})
	url.WriteToDbBuffer()
	url.SaveToDB()
}

func increaseNonce(db interfaces.Datastore, txs []*cmntyp.StandardMessage) {
	url := ccurl.NewConcurrentUrl(db)
	// stateDB := adaptor.NewStateDBV2(nil, db, url)
	api := ccapi.NewAPI(url)
	stateDB := eth.NewImplStateDB(api)
	stateDB.PrepareFormer(evmCommon.Hash{}, evmCommon.Hash{}, 0)
	for i := range txs {
		address := evmCommon.BytesToAddress(txs[i].Native.From.Bytes())
		stateDB.SetNonce(address, 0)
	}
	_, transitions := url.ExportAll()
	url.Import(transitions)
	url.Sort()
	url.Finalize([]uint32{0})
	url.WriteToDbBuffer()
	url.SaveToDB()
}

func genUncheckedTxs(from, to int) []*cmntyp.StandardMessage {
	txs := make([]*cmntyp.StandardMessage, to-from)
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
		txs[i-from] = &cmntyp.StandardMessage{
			TxHash: hash,
			Native: &msg,
		}
	}
	return txs
}

func genCheckedTxs(from, to int, nonce uint64, gasPrice uint64) []*cmntyp.StandardMessage {
	txs := make([]*cmntyp.StandardMessage, to-from)
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
		txs[i-from] = &cmntyp.StandardMessage{
			TxHash: hash,
			Native: &msg,
		}
	}
	return txs
}
