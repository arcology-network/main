//go:build !CI

package boot

import (
	"math/big"
	"math/rand"
	"testing"
	"time"

	ethcommon "github.com/arcology-network/3rd-party/eth/common"
	cmntypes "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	intf "github.com/arcology-network/component-lib/interface"
	"github.com/arcology-network/component-lib/mock/kafka"
	"github.com/arcology-network/component-lib/mock/rpc"
	urlcommon "github.com/arcology-network/concurrenturl/v2/common"
	urltypes "github.com/arcology-network/concurrenturl/v2/type"
	"github.com/arcology-network/concurrenturl/v2/type/commutative"
	"github.com/arcology-network/main/config"
)

func TestBootstrapCase1(t *testing.T) {
	hash1 := ethcommon.BytesToHash([]byte("hash1"))
	hash2 := ethcommon.BytesToHash([]byte("hash2"))
	hash3 := ethcommon.BytesToHash([]byte("hash3"))
	hash4 := ethcommon.BytesToHash([]byte("hash4"))
	response := runTestCase(
		t,
		[][]*cmntypes.TxElement{{createTxElement(hash1, 0, 1)}, {createTxElement(hash2, 0, 2)}, {createTxElement(hash3, 0, 3)}, {createTxElement(hash4, 0, 4)}},
		newAccessRecords(hash1, 1,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-50))),
		),
		newAccessRecords(hash2, 2,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-100))),
		),
		newAccessRecords(hash3, 3,
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-50))),
		),
		newAccessRecords(hash4, 4,
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-50))),
		),
	)
	t.Log(response)
	if len(response.ConflictedList) != 2 || len(response.CPairLeft) != 1 || len(response.CPairRight) != 1 {
		t.Fail()
	}
}

func TestBootstrapCase2(t *testing.T) {
	hashes := []ethcommon.Hash{
		ethcommon.BytesToHash([]byte("hash1")),
		ethcommon.BytesToHash([]byte("hash2")),
		ethcommon.BytesToHash([]byte("hash3")),
		ethcommon.BytesToHash([]byte("hash4")),
		ethcommon.BytesToHash([]byte("hash5")),
		ethcommon.BytesToHash([]byte("hash6")),
		ethcommon.BytesToHash([]byte("hash7")),
		ethcommon.BytesToHash([]byte("hash8")),
		ethcommon.BytesToHash([]byte("hash9")),
		ethcommon.BytesToHash([]byte("hash10")),
		ethcommon.BytesToHash([]byte("hash11")),
		ethcommon.BytesToHash([]byte("hash12")),
	}
	response := runTestCase(
		t,
		[][]*cmntypes.TxElement{
			{createTxElement(hashes[0], 0, 1), createTxElement(hashes[1], 0, 2), createTxElement(hashes[2], 1, 3)},
			{createTxElement(hashes[3], 0, 4), createTxElement(hashes[4], 0, 5), createTxElement(hashes[5], 1, 6)},
			{createTxElement(hashes[6], 0, 7), createTxElement(hashes[7], 0, 8), createTxElement(hashes[8], 1, 9)},
			{createTxElement(hashes[9], 0, 10), createTxElement(hashes[10], 0, 11), createTxElement(hashes[11], 1, 12)},
		},
		newAccessRecords(hashes[0], 1,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10))),
		),
		newAccessRecords(hashes[1], 2,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key2", 0, 1, true, false, nil),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10))),
		),
		newAccessRecords(hashes[2], 3,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10))),
		),
		newAccessRecords(hashes[3], 4,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 1, 0, true, false, nil),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10))),
		),
		newAccessRecords(hashes[4], 5,
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10))),
		),
		newAccessRecords(hashes[5], 6,
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-10))),
		),
		newAccessRecords(hashes[6], 7,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map2/key1", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[7], 8,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map2/key2", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[8], 9,
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-50))),
		),
		newAccessRecords(hashes[9], 10,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map3/key1", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[10], 11,
			newAccess(urlcommon.NoncommutativeBytes, "blcc://eth1.0/accounts/Alice/storage/containers/map3/key2", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[11], 12,
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(100), new(big.Int).SetInt64(-50))),
		),
	)
	t.Log(response)
	if len(response.ConflictedList) != 6 || len(response.CPairLeft) != 2 || len(response.CPairRight) != 2 {
		t.Fail()
	}
}

func TestDetectConflictPerf(t *testing.T) {
	// TestBootstrapCase1(t)
	// TestBootstrapCase1(t)
	NTXS := 2000
	hashes := make([]ethcommon.Hash, NTXS)
	for i := 0; i < NTXS; i++ {
		hashes[i] = ethcommon.BytesToHash([]byte(RandStringRunes(32)))
	}
	addresses := make([]ethcommon.Address, NTXS*2)
	for i := 0; i < NTXS*2; i++ {
		addresses[i] = ethcommon.BytesToAddress([]byte(RandStringRunes(20)))
	}
	coinbase := ethcommon.BytesToHash([]byte(RandStringRunes(20)))
	groups := make([][]*cmntypes.TxElement, NTXS)
	for i := 0; i < NTXS; i++ {
		groups[i] = []*cmntypes.TxElement{createTxElement(hashes[i], 0, uint32(i+1))}
	}
	records := make([]*accessRecords, NTXS)
	for i := 0; i < NTXS; i++ {
		records[i] = newAccessRecords(
			hashes[i],
			uint32(i+1),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/"+addresses[i*2].Hex()+"/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(1000000000), new(big.Int).SetInt64(-2))),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/"+addresses[i*2+1].Hex()+"/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(1000000000), new(big.Int).SetInt64(1))),
			newAccess(urlcommon.CommutativeBalance, "blcc://eth1.0/accounts/"+coinbase.Hex()+"/balance", 0, 1, true, true, commutative.NewBalance(new(big.Int).SetInt64(1000000000), new(big.Int).SetInt64(1))),
			newAccess(urlcommon.NoncommutativeInt64, "blcc://eth1.0/accounts/"+addresses[i*2].Hex()+"/nonce", 0, 1, true, true, commutative.NewInt64(0, 1)),
			newAccess(urlcommon.CommutativeMeta, "blcc://eth1.0/accounts/", 1, 0, true, false, nil),
		)
	}

	begin := time.Now()
	response := runTestCase(
		t,
		groups,
		records...,
	)
	t.Log(response)
	t.Log(time.Since(begin))
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

type access struct {
	vType     uint8
	path      string
	reads     uint32
	writes    uint32
	preexists bool
	composite bool
	value     interface{}
}

func newAccess(vType uint8, path string, reads uint32, writes uint32, preexists bool, composite bool, value interface{}) *access {
	return &access{
		vType:     vType,
		path:      path,
		reads:     reads,
		writes:    writes,
		preexists: preexists,
		composite: composite,
		value:     value,
	}
}

type accessRecords struct {
	hash     ethcommon.Hash
	id       uint32
	accesses []*access
}

func newAccessRecords(hash ethcommon.Hash, id uint32, accesses ...*access) *accessRecords {
	accessRecords := &accessRecords{
		hash: hash,
		id:   id,
	}
	accessRecords.accesses = append(accessRecords.accesses, accesses...)
	return accessRecords
}

func createTxElement(hash ethcommon.Hash, batch uint64, tx uint32) *cmntypes.TxElement {
	return &cmntypes.TxElement{
		TxHash:  &hash,
		Batchid: batch,
		Txid:    tx,
	}
}

func runTestCase(t *testing.T, txGroups [][]*cmntypes.TxElement, records ...*accessRecords) *cmntypes.ArbitratorResponse {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		t.Log(r)
	// 	}
	// }()
	config.DownloaderCreator = kafka.NewDownloaderCreator(t)
	config.UploaderCreator = kafka.NewUploaderCreator(t)
	intf.RPCCreator = rpc.NewRPCServerInitializer(t)

	globalConfig := config.LoadGlobalConfig("../config/global.json")
	kafkaConfig := config.LoadKafkaConfig("../config/kafka.json")
	appConfig := config.LoadAppConfig("../modules/arbitrator/arbitrator.json")
	brk, _, _ := initApp(globalConfig, kafkaConfig, appConfig)

	broker := &actor.MessageWrapper{
		MsgBroker:      brk,
		LatestMessage:  actor.NewMessage(),
		WorkThreadName: "unittester",
	}

	var txAccessRecords cmntypes.TxAccessRecordSet
	for _, record := range records {
		univalues := urltypes.Univalues{}
		for _, a := range record.accesses {
			univalues = append(univalues, urltypes.CreateUnivalueForTest(urlcommon.VARIATE_TRANSITIONS, a.vType, record.id, a.path, a.reads, a.writes, a.value, a.preexists, a.composite))
		}
		txAccessRecords = append(txAccessRecords, &cmntypes.TxAccessRecords{
			Hash:     string(record.hash.Bytes()),
			ID:       record.id,
			Accesses: univalues.EncodeV2(),
		})
	}

	broker.Send(actor.MsgTxAccessRecords, &txAccessRecords)
	// for i := 0; i < len(txAccessRecords)/500; i++ {
	// 	data := txAccessRecords[i*500 : (i+1)*500]
	// 	kafka2.Receive(&actor.Message{
	// 		Name: actor.MsgTxAccessRecords,
	// 		Data: &data,
	// 	})
	// }

	response := cmntypes.ArbitratorResponse{}
	intf.Router.Call("arbitrator", "Arbitrate", &actor.Message{
		Data: &cmntypes.ArbitratorRequest{TxsListGroup: txGroups},
	}, &response)

	if len(response.ConflictedList) == 0 && len(response.CPairLeft) != 0 {
		t.Log("SOMETHING WEIRD HAPPENED.")
		t.Log(response)
		intf.Router.Call("arbitrator", "Arbitrate", &actor.Message{
			Data: &cmntypes.ArbitratorRequest{TxsListGroup: txGroups},
		}, &response)
		t.Fail()
	}

	// Clear arbitrator.
	broker.Send(actor.MsgBlockCompleted, nil)
	return &response
}
