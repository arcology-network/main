//go:build !CI

package boot

import (
	"math/rand"
	"testing"
	"time"

	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/noncommutative"
	univaluepk "github.com/arcology-network/concurrenturl/univalue"
	eushared "github.com/arcology-network/eu/shared"
	"github.com/arcology-network/main/config"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
	"github.com/arcology-network/streamer/mock/kafka"
	"github.com/arcology-network/streamer/mock/rpc"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

func TestBootstrapCase1(t *testing.T) {
	hash1 := evmCommon.BytesToHash([]byte("hash1"))
	hash2 := evmCommon.BytesToHash([]byte("hash2"))
	hash3 := evmCommon.BytesToHash([]byte("hash3"))
	hash4 := evmCommon.BytesToHash([]byte("hash4"))

	response := runTestCase(
		t,
		[][]*mtypes.TxElement{{createTxElement(hash1, 0, 1)}, {createTxElement(hash2, 0, 2)}, {createTxElement(hash3, 0, 3)}, {createTxElement(hash4, 0, 4)}},
		newAccessRecords(hash1, 1,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 50, false, nil, nil)),
		),
		newAccessRecords(hash2, 2,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 100, false, nil, nil)),
		),
		newAccessRecords(hash3, 3,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 50, false, nil, nil)),
		),
		newAccessRecords(hash4, 4,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 50, false, nil, nil)),
		),
	)
	t.Log(response)
	if len(response.ConflictedList) != 2 || len(response.CPairLeft) != 1 || len(response.CPairRight) != 1 {
		t.Fail()
	}
}

func TestBootstrapCase2(t *testing.T) {
	hashes := []evmCommon.Hash{
		evmCommon.BytesToHash([]byte("hash1")),
		evmCommon.BytesToHash([]byte("hash2")),
		evmCommon.BytesToHash([]byte("hash3")),
		evmCommon.BytesToHash([]byte("hash4")),
		evmCommon.BytesToHash([]byte("hash5")),
		evmCommon.BytesToHash([]byte("hash6")),
		evmCommon.BytesToHash([]byte("hash7")),
		evmCommon.BytesToHash([]byte("hash8")),
		evmCommon.BytesToHash([]byte("hash9")),
		evmCommon.BytesToHash([]byte("hash10")),
		evmCommon.BytesToHash([]byte("hash11")),
		evmCommon.BytesToHash([]byte("hash12")),
	}
	response := runTestCase(
		t,
		[][]*mtypes.TxElement{
			{createTxElement(hashes[0], 0, 1), createTxElement(hashes[1], 0, 2), createTxElement(hashes[2], 1, 3)},
			{createTxElement(hashes[3], 0, 4), createTxElement(hashes[4], 0, 5), createTxElement(hashes[5], 1, 6)},
			{createTxElement(hashes[6], 0, 7), createTxElement(hashes[7], 0, 8), createTxElement(hashes[8], 1, 9)},
			{createTxElement(hashes[9], 0, 10), createTxElement(hashes[10], 0, 11), createTxElement(hashes[11], 1, 12)},
		},
		newAccessRecords(hashes[0], 1,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 10, false, nil, nil)),
		),
		newAccessRecords(hashes[1], 2,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key2", 0, 1, true, false, nil),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 10, false, nil, nil)),
		),
		newAccessRecords(hashes[2], 3,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 0, 1, true, false, nil),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 10, false, nil, nil)),
		),
		newAccessRecords(hashes[3], 4,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map1/key1", 1, 0, true, false, nil),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 10, false, nil, nil)),
		),
		newAccessRecords(hashes[4], 5,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 10, false, nil, nil)),
		),
		newAccessRecords(hashes[5], 6,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 10, false, nil, nil)),
		),
		newAccessRecords(hashes[6], 7,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/storage/containers/map2/key1", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[7], 8,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/storage/containers/map2/key2", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[8], 9,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 50, false, nil, nil)),
		),
		newAccessRecords(hashes[9], 10,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map3/key1", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[10], 11,
			newAccess(noncommutative.BYTES, "blcc://eth1.0/accounts/Alice/storage/containers/map3/key2", 0, 1, true, false, nil),
		),
		newAccessRecords(hashes[11], 12,
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/Alice/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(100, 50, false, nil, nil)),
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
	hashes := make([]evmCommon.Hash, NTXS)
	for i := 0; i < NTXS; i++ {
		hashes[i] = evmCommon.BytesToHash([]byte(RandStringRunes(32)))
	}
	addresses := make([]evmCommon.Address, NTXS*2)
	for i := 0; i < NTXS*2; i++ {
		addresses[i] = evmCommon.BytesToAddress([]byte(RandStringRunes(20)))
	}
	coinbase := evmCommon.BytesToHash([]byte(RandStringRunes(20)))
	groups := make([][]*mtypes.TxElement, NTXS)
	for i := 0; i < NTXS; i++ {
		groups[i] = []*mtypes.TxElement{createTxElement(hashes[i], 0, uint32(i+1))}
	}
	records := make([]*accessRecords, NTXS)
	for i := 0; i < NTXS; i++ {
		records[i] = newAccessRecords(
			hashes[i],
			uint32(i+1),

			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/"+addresses[i*2].Hex()+"/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(1000000000, 2, false, nil, nil)),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/"+addresses[i*2+1].Hex()+"/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(1000000000, 1, true, nil, nil)),
			newAccess(commutative.UINT256, "blcc://eth1.0/accounts/"+coinbase.Hex()+"/balance", 0, 1, true, true, commutative.NewUnboundedU256().(*commutative.U256).New(1000000000, 1, true, nil, nil)),
			newAccess(commutative.UINT64, "blcc://eth1.0/accounts/"+addresses[i*2].Hex()+"/nonce", 0, 1, true, true, commutative.NewInt64(0, 1)),
			newAccess(commutative.PATH, "blcc://eth1.0/accounts/", 1, 0, true, false, nil),
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
	hash     evmCommon.Hash
	id       uint32
	accesses []*access
}

func newAccessRecords(hash evmCommon.Hash, id uint32, accesses ...*access) *accessRecords {
	accessRecords := &accessRecords{
		hash: hash,
		id:   id,
	}
	accessRecords.accesses = append(accessRecords.accesses, accesses...)
	return accessRecords
}

func createTxElement(hash evmCommon.Hash, batch uint64, tx uint32) *mtypes.TxElement {
	return &mtypes.TxElement{
		TxHash:  &hash,
		Batchid: batch,
		Txid:    tx,
	}
}

func runTestCase(t *testing.T, txGroups [][]*mtypes.TxElement, records ...*accessRecords) *mtypes.ArbitratorResponse {
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

	var txAccessRecords eushared.TxAccessRecordSet
	for _, record := range records {
		univalues := []*univaluepk.Univalue{}
		for _, a := range record.accesses {

			univalues = append(univalues, univaluepk.NewUnivalue(record.id, a.path, a.reads, a.writes, 0, a.value, nil))
		}
		txAccessRecords = append(txAccessRecords, &eushared.TxAccessRecords{
			Hash:     string(record.hash.Bytes()),
			ID:       record.id,
			Accesses: univaluepk.Univalues(univalues).Encode(), //univaluepk.UnivaluesEncode(univalues),
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

	response := mtypes.ArbitratorResponse{}
	intf.Router.Call("arbitrator", "Arbitrate", &actor.Message{
		Data: &mtypes.ArbitratorRequest{TxsListGroup: txGroups},
	}, &response)

	if len(response.ConflictedList) == 0 && len(response.CPairLeft) != 0 {
		t.Log("SOMETHING WEIRD HAPPENED.")
		t.Log(response)
		intf.Router.Call("arbitrator", "Arbitrate", &actor.Message{
			Data: &mtypes.ArbitratorRequest{TxsListGroup: txGroups},
		}, &response)
		t.Fail()
	}

	// Clear arbitrator.
	broker.Send(actor.MsgBlockCompleted, nil)
	return &response
}
