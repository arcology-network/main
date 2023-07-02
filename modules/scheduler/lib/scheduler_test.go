package lib

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/core"
)

func TestScheduler(t *testing.T) {

	from0 := evmCommon.BytesToAddress([]byte{0, 0, 0, 1, 4, 5, 6, 7, 8})
	to0 := evmCommon.BytesToAddress([]byte{11, 12, 13, 14})
	from1 := evmCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9})
	to1 := evmCommon.BytesToAddress([]byte{11, 12, 13})
	from2 := evmCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9, 10, 11, 12})
	to2 := evmCommon.BytesToAddress([]byte{11, 12, 13, 14, 15, 16, 17})
	from3 := evmCommon.BytesToAddress([]byte{1, 1, 1, 5, 6, 7, 8, 9, 10, 11, 12})
	to3 := evmCommon.BytesToAddress([]byte{1, 1, 1, 11, 12, 13, 14, 15, 16, 17})
	from4 := evmCommon.BytesToAddress([]byte{2, 2, 2, 5, 6, 7, 8, 9, 10, 11, 12})
	//to4 := evmCommon.BytesToAddress([]byte{2, 2, 2, 11, 12, 13, 14, 15, 16, 17})
	from5 := evmCommon.BytesToAddress([]byte{3, 3, 3, 5, 6, 7, 8, 9, 10, 11, 12})
	to5 := evmCommon.BytesToAddress([]byte{3, 3, 3, 11, 12, 13, 14, 15, 16, 17})
	ethMsg_serial_0 := core.NewMessage(from0, &to0, 1, big.NewInt(int64(1)), 100, big.NewInt(int64(8)), []byte{1, 2, 3}, nil, false)
	ethMsg_serial_1 := core.NewMessage(from1, &to1, 3, big.NewInt(int64(100)), 200, big.NewInt(int64(9)), []byte{4, 5, 6}, nil, false)
	ethMsg_serial_2 := core.NewMessage(from2, &to2, 2, big.NewInt(int64(121)), 300, big.NewInt(int64(10)), []byte{7, 8, 9}, nil, false)
	ethMsg_serial_3 := core.NewMessage(from3, &to3, 4, big.NewInt(int64(2000)), 400, big.NewInt(int64(11)), []byte{10, 11, 12}, nil, false)
	ethMsg_serial_4 := core.NewMessage(from4, nil, 5, big.NewInt(int64(321)), 500, big.NewInt(int64(12)), []byte{13, 14, 15}, nil, false)
	ethMsg_serial_5 := core.NewMessage(from5, &to5, 6, big.NewInt(int64(134)), 600, big.NewInt(int64(13)), []byte{16, 17, 18}, nil, false)
	hash1 := types.RlpHash(&ethMsg_serial_0)
	hash2 := types.RlpHash(&ethMsg_serial_1)
	hash3 := types.RlpHash(&ethMsg_serial_2)
	hash4 := types.RlpHash(&ethMsg_serial_3)
	hash5 := types.RlpHash(&ethMsg_serial_4)
	hash6 := types.RlpHash(&ethMsg_serial_5)
	fmt.Printf("hash1=%v\n", hash1)
	fmt.Printf("hash2=%v\n", hash2)
	fmt.Printf("hash3=%v\n", hash3)
	fmt.Printf("hash4=%v\n", hash4)
	fmt.Printf("hash5=%v\n", hash5)
	fmt.Printf("hash6=%v\n", hash6)

	msgs := []*types.StandardMessage{
		{Source: 0, Native: &ethMsg_serial_0, TxHash: hash1},
		{Source: 1, Native: &ethMsg_serial_1, TxHash: hash2},
		{Source: 1, Native: &ethMsg_serial_2, TxHash: hash3},
		{Source: 1, Native: &ethMsg_serial_3, TxHash: hash4},
		{Source: 1, Native: &ethMsg_serial_4, TxHash: hash5},
		{Source: 1, Native: &ethMsg_serial_5, TxHash: hash6},
	}
	callees, txids := GetCallees(msgs)
	fmt.Printf("callees=%v,txids=%v\n", callees, txids)

	result_callees := []byte{48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 66, 48, 99, 48, 68, 48, 101, 0, 0, 0, 0, 0, 0, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 98, 48, 67, 48, 100, 0, 0, 0, 0, 0, 0, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 66, 48, 67, 48, 100, 48, 69, 48, 70, 49, 48, 49, 49, 0, 0, 0, 0, 0, 0, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 49, 48, 49, 48, 49, 48, 98, 48, 67, 48, 68, 48, 101, 48, 70, 49, 48, 49, 49, 0, 0, 0, 0, 0, 0, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 0, 0, 0, 0, 0, 0, 0, 0, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 51, 48, 51, 48, 51, 48, 98, 48, 99, 48, 68, 48, 101, 48, 102, 49, 48, 49, 49, 0, 0, 0, 0, 0, 0, 0, 0}

	if !reflect.DeepEqual(callees, result_callees) {
		t.Error("GetCallees err in field callees!", callees, result_callees)
	}
	result_txids := []uint32{0, 1, 2, 3, 4, 5}
	if !reflect.DeepEqual(txids, result_txids) {
		t.Error("GetCallees err in field txids!", txids, result_txids)
	}

	orderIDs := []uint32{5, 4, 2, 3, 1, 0}
	branches := []uint32{0, 1, 1, 1, 0, 1}
	generations := []uint32{0, 0, 0, 0, 1, 1}

	sequencess := ParseResult(msgs, orderIDs, branches, generations)

	for _, list := range sequencess {
		fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n")
		for _, sequence := range list {
			fmt.Printf("sequence.Parallel=%v\n", sequence.Parallel)
			for _, msg := range sequence.Msgs {
				fmt.Printf("msg.TxHash=%v\n", msg.TxHash)
			}
		}
	}
}

func TestSchedule(t *testing.T) {

	from0 := evmCommon.BytesToAddress([]byte{0, 0, 0, 1, 4, 5, 6, 7, 8})
	to0 := evmCommon.BytesToAddress([]byte{11, 12, 13, 14})
	from1 := evmCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9})
	to1 := evmCommon.BytesToAddress([]byte{11, 12, 13})
	from2 := evmCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9, 10, 11, 12})
	//to2 := evmCommon.BytesToAddress([]byte{11, 12, 13, 14, 15, 16, 17})
	from3 := evmCommon.BytesToAddress([]byte{1, 1, 1, 5, 6, 7, 8, 9, 10, 11, 12})
	// to3 := evmCommon.BytesToAddress([]byte{1, 1, 1, 11, 12, 13, 14, 15, 16, 17})
	from4 := evmCommon.BytesToAddress([]byte{2, 2, 2, 5, 6, 7, 8, 9, 10, 11, 12})
	//to4 := evmCommon.BytesToAddress([]byte{2, 2, 2, 11, 12, 13, 14, 15, 16, 17})
	from5 := evmCommon.BytesToAddress([]byte{3, 3, 3, 5, 6, 7, 8, 9, 10, 11, 12})
	//to5 := evmCommon.BytesToAddress([]byte{3, 3, 3, 11, 12, 13, 14, 15, 16, 17})
	ethMsg_serial_0 := core.NewMessage(from0, &to0, 1, big.NewInt(int64(1)), 100, big.NewInt(int64(8)), []byte{1, 2, 3}, nil, false)
	ethMsg_serial_1 := core.NewMessage(from1, &to0, 3, big.NewInt(int64(100)), 200, big.NewInt(int64(9)), []byte{4, 5, 6}, nil, false)
	ethMsg_serial_2 := core.NewMessage(from2, &to0, 2, big.NewInt(int64(121)), 300, big.NewInt(int64(10)), []byte{7, 8, 9}, nil, false)
	ethMsg_serial_3 := core.NewMessage(from3, &to1, 4, big.NewInt(int64(2000)), 400, big.NewInt(int64(11)), []byte{10, 11, 12}, nil, false)
	ethMsg_serial_4 := core.NewMessage(from4, &to1, 5, big.NewInt(int64(321)), 500, big.NewInt(int64(12)), []byte{13, 14, 15}, nil, false)
	ethMsg_serial_5 := core.NewMessage(from5, &to1, 6, big.NewInt(int64(134)), 600, big.NewInt(int64(13)), []byte{16, 17, 18}, nil, false)
	hash1 := types.RlpHash(&ethMsg_serial_0)
	hash2 := types.RlpHash(&ethMsg_serial_1)
	hash3 := types.RlpHash(&ethMsg_serial_2)
	hash4 := types.RlpHash(&ethMsg_serial_3)
	hash5 := types.RlpHash(&ethMsg_serial_4)
	hash6 := types.RlpHash(&ethMsg_serial_5)
	fmt.Printf("hash1=%v\n", hash1)
	fmt.Printf("hash2=%v\n", hash2)
	fmt.Printf("hash3=%v\n", hash3)
	fmt.Printf("hash4=%v\n", hash4)
	fmt.Printf("hash5=%v\n", hash5)
	fmt.Printf("hash6=%v\n", hash6)

	msgs := []*types.StandardMessage{
		{Source: 0, Native: &ethMsg_serial_0, TxHash: hash1},
		{Source: 1, Native: &ethMsg_serial_1, TxHash: hash2},
		//{Source: 1, Native: &ethMsg_serial_2, TxHash: hash3},
		{Source: 1, Native: &ethMsg_serial_3, TxHash: hash4},
		{Source: 1, Native: &ethMsg_serial_4, TxHash: hash5},
		//{Source: 1, Native: &ethMsg_serial_5, TxHash: hash6},
	}

	scheduler, err := Start("history.binn")
	fmt.Printf(">>>>>>>>start log=%v\n", err)

	log := scheduler.Update([]*evmCommon.Address{&to0, &to1}, []*evmCommon.Address{&to0, &to1})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log=%v\n", log)
	}
	sequencess, log := scheduler.Schedule(msgs, 0)

	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Schedule log=%v\n", log)
	}

	for _, list := range sequencess {
		fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n")
		for _, sequence := range list {
			fmt.Printf("sequence.Parallel=%v\n", sequence.Parallel)
			for _, msg := range sequence.Msgs {
				fmt.Printf("msg.TxHash=%v\n", msg.TxHash)
			}
		}
	}
}

func TestScheduleProfermance(t *testing.T) {
	ds_address := evmCommon.HexToAddress("842d4bfdb1904503ac152483527f338cd5d9bcba")
	kittyCore := evmCommon.HexToAddress("b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f")
	transfer := evmCommon.HexToAddress("b1e0e9e68297aae01347f6ce0ff21d5f72d3fa2a")

	from := evmCommon.HexToAddress("001111e68297aae01347f6ce0ff21d5f72d3fa2a")

	msgs := make([]*types.StandardMessage, 0, 50000)

	ds_token_count := 16568
	start := 0
	for i := 0; i < ds_token_count; i++ {
		idx := start + i + 1
		native := core.NewMessage(from, &ds_address, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, nil, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: types.RlpHash(&native),
		})
	}
	pk_count := 16638
	start = ds_token_count
	for i := 0; i < pk_count; i++ {
		idx := start + i + 1
		native := core.NewMessage(from, &kittyCore, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, nil, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: types.RlpHash(&native),
		})
	}
	transfer_count := 16794
	start = ds_token_count + pk_count
	for i := 0; i < transfer_count; i++ {
		idx := start + i + 1
		native := core.NewMessage(from, &transfer, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, nil, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: types.RlpHash(&native),
		})
	}
	scheduler, err := Start("history.binn")
	fmt.Printf(">>>>>>>>start log=%v\n", err)

	log := scheduler.Update([]*evmCommon.Address{&ds_address, &kittyCore}, []*evmCommon.Address{&ds_address, &kittyCore})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log=%v\n", log)
	}

	starttime := time.Now()
	sequencess, log := scheduler.Schedule(msgs, 0)
	fmt.Printf(">>>>>>>>Schedule time=%v\n", time.Since(starttime))
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Schedule log=%v\n", log)
	}

	for _, list := range sequencess {
		fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>list=%v\n", len(list))
		for _, sequence := range list {
			fmt.Printf("sequence.Parallel=%v,size=%v\n", sequence.Parallel, len(sequence.Msgs))
		}
	}
}

func TestScheduleCase(t *testing.T) {
	addr1 := evmCommon.HexToAddress("fbc451fbd7e17a1e7b18347337657c1f2c52b631")
	addr2 := evmCommon.HexToAddress("2249977665260a63307cf72a4d65385cc0817cb5")
	addr3 := evmCommon.HexToAddress("1642dd5c38642f91e4aa0025978b572fe30ed89d")
	addr4 := evmCommon.HexToAddress("f5848bbe1a6ed5c4e9a5ba10c088bc28062a5d13")

	from := evmCommon.HexToAddress("001111e68297aae01347f6ce0ff21d5f72d3fa2a")

	msgs := make([]*types.StandardMessage, 0, 50000)

	ds_token_count := 0
	start := 0
	for i := 0; i < ds_token_count; i++ {
		idx := start + i + 1
		native := core.NewMessage(from, &addr1, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, nil, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: types.RlpHash(&native),
		})
	}
	pk_count := 10
	start = ds_token_count
	for i := 0; i < pk_count; i++ {
		idx := start + i + 1
		native := core.NewMessage(from, &addr2, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, nil, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: types.RlpHash(&native),
		})
	}
	transfer_count := 10
	start = ds_token_count + pk_count
	for i := 0; i < transfer_count; i++ {
		idx := start + i + 1
		native := core.NewMessage(from, &addr3, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, nil, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: types.RlpHash(&native),
		})
	}
	scheduler, err := Start("history.binn")
	fmt.Printf(">>>>>>>>start log=%v\n", err)

	log := scheduler.Update([]*evmCommon.Address{&addr1}, []*evmCommon.Address{&addr1})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log1=%v\n", log)
	}
	log = scheduler.Update([]*evmCommon.Address{&addr2}, []*evmCommon.Address{&addr2})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log2=%v\n", log)
	}
	log = scheduler.Update([]*evmCommon.Address{&addr3}, []*evmCommon.Address{&addr3})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log3=%v\n", log)
	}
	log = scheduler.Update([]*evmCommon.Address{&addr4}, []*evmCommon.Address{&addr4})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log4=%v\n", log)
	}

	starttime := time.Now()
	sequencess, log := scheduler.Schedule(msgs, 0)
	fmt.Printf(">>>>>>>>Schedule time=%v\n", time.Since(starttime))
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Schedule log=%v\n", log)
	}

	for _, list := range sequencess {
		fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>list=%v\n", len(list))
		for _, sequence := range list {
			fmt.Printf("sequence.Parallel=%v,size=%v\n", sequence.Parallel, len(sequence.Msgs))
		}
	}
}
