package lib

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/types"
)

func TestScheduler(t *testing.T) {

	from0 := ethCommon.BytesToAddress([]byte{0, 0, 0, 1, 4, 5, 6, 7, 8})
	to0 := ethCommon.BytesToAddress([]byte{11, 12, 13, 14})
	from1 := ethCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9})
	to1 := ethCommon.BytesToAddress([]byte{11, 12, 13})
	from2 := ethCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9, 10, 11, 12})
	to2 := ethCommon.BytesToAddress([]byte{11, 12, 13, 14, 15, 16, 17})
	from3 := ethCommon.BytesToAddress([]byte{1, 1, 1, 5, 6, 7, 8, 9, 10, 11, 12})
	to3 := ethCommon.BytesToAddress([]byte{1, 1, 1, 11, 12, 13, 14, 15, 16, 17})
	from4 := ethCommon.BytesToAddress([]byte{2, 2, 2, 5, 6, 7, 8, 9, 10, 11, 12})
	//to4 := ethCommon.BytesToAddress([]byte{2, 2, 2, 11, 12, 13, 14, 15, 16, 17})
	from5 := ethCommon.BytesToAddress([]byte{3, 3, 3, 5, 6, 7, 8, 9, 10, 11, 12})
	to5 := ethCommon.BytesToAddress([]byte{3, 3, 3, 11, 12, 13, 14, 15, 16, 17})
	ethMsg_serial_0 := ethTypes.NewMessage(from0, &to0, 1, big.NewInt(int64(1)), 100, big.NewInt(int64(8)), []byte{1, 2, 3}, false)
	ethMsg_serial_1 := ethTypes.NewMessage(from1, &to1, 3, big.NewInt(int64(100)), 200, big.NewInt(int64(9)), []byte{4, 5, 6}, false)
	ethMsg_serial_2 := ethTypes.NewMessage(from2, &to2, 2, big.NewInt(int64(121)), 300, big.NewInt(int64(10)), []byte{7, 8, 9}, false)
	ethMsg_serial_3 := ethTypes.NewMessage(from3, &to3, 4, big.NewInt(int64(2000)), 400, big.NewInt(int64(11)), []byte{10, 11, 12}, false)
	ethMsg_serial_4 := ethTypes.NewMessage(from4, nil, 5, big.NewInt(int64(321)), 500, big.NewInt(int64(12)), []byte{13, 14, 15}, false)
	ethMsg_serial_5 := ethTypes.NewMessage(from5, &to5, 6, big.NewInt(int64(134)), 600, big.NewInt(int64(13)), []byte{16, 17, 18}, false)
	hash1 := ethCommon.RlpHash(&ethMsg_serial_0)
	hash2 := ethCommon.RlpHash(&ethMsg_serial_1)
	hash3 := ethCommon.RlpHash(&ethMsg_serial_2)
	hash4 := ethCommon.RlpHash(&ethMsg_serial_3)
	hash5 := ethCommon.RlpHash(&ethMsg_serial_4)
	hash6 := ethCommon.RlpHash(&ethMsg_serial_5)
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

	from0 := ethCommon.BytesToAddress([]byte{0, 0, 0, 1, 4, 5, 6, 7, 8})
	to0 := ethCommon.BytesToAddress([]byte{11, 12, 13, 14})
	from1 := ethCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9})
	to1 := ethCommon.BytesToAddress([]byte{11, 12, 13})
	from2 := ethCommon.BytesToAddress([]byte{0, 0, 0, 5, 6, 7, 8, 9, 10, 11, 12})
	//to2 := ethCommon.BytesToAddress([]byte{11, 12, 13, 14, 15, 16, 17})
	from3 := ethCommon.BytesToAddress([]byte{1, 1, 1, 5, 6, 7, 8, 9, 10, 11, 12})
	// to3 := ethCommon.BytesToAddress([]byte{1, 1, 1, 11, 12, 13, 14, 15, 16, 17})
	from4 := ethCommon.BytesToAddress([]byte{2, 2, 2, 5, 6, 7, 8, 9, 10, 11, 12})
	//to4 := ethCommon.BytesToAddress([]byte{2, 2, 2, 11, 12, 13, 14, 15, 16, 17})
	from5 := ethCommon.BytesToAddress([]byte{3, 3, 3, 5, 6, 7, 8, 9, 10, 11, 12})
	//to5 := ethCommon.BytesToAddress([]byte{3, 3, 3, 11, 12, 13, 14, 15, 16, 17})
	ethMsg_serial_0 := ethTypes.NewMessage(from0, &to0, 1, big.NewInt(int64(1)), 100, big.NewInt(int64(8)), []byte{1, 2, 3}, false)
	ethMsg_serial_1 := ethTypes.NewMessage(from1, &to0, 3, big.NewInt(int64(100)), 200, big.NewInt(int64(9)), []byte{4, 5, 6}, false)
	ethMsg_serial_2 := ethTypes.NewMessage(from2, &to0, 2, big.NewInt(int64(121)), 300, big.NewInt(int64(10)), []byte{7, 8, 9}, false)
	ethMsg_serial_3 := ethTypes.NewMessage(from3, &to1, 4, big.NewInt(int64(2000)), 400, big.NewInt(int64(11)), []byte{10, 11, 12}, false)
	ethMsg_serial_4 := ethTypes.NewMessage(from4, &to1, 5, big.NewInt(int64(321)), 500, big.NewInt(int64(12)), []byte{13, 14, 15}, false)
	ethMsg_serial_5 := ethTypes.NewMessage(from5, &to1, 6, big.NewInt(int64(134)), 600, big.NewInt(int64(13)), []byte{16, 17, 18}, false)
	hash1 := ethCommon.RlpHash(&ethMsg_serial_0)
	hash2 := ethCommon.RlpHash(&ethMsg_serial_1)
	hash3 := ethCommon.RlpHash(&ethMsg_serial_2)
	hash4 := ethCommon.RlpHash(&ethMsg_serial_3)
	hash5 := ethCommon.RlpHash(&ethMsg_serial_4)
	hash6 := ethCommon.RlpHash(&ethMsg_serial_5)
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

	log := scheduler.Update([]*ethCommon.Address{&to0, &to1}, []*ethCommon.Address{&to0, &to1})
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
	ds_address := ethCommon.HexToAddress("842d4bfdb1904503ac152483527f338cd5d9bcba")
	kittyCore := ethCommon.HexToAddress("b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f")
	transfer := ethCommon.HexToAddress("b1e0e9e68297aae01347f6ce0ff21d5f72d3fa2a")

	from := ethCommon.HexToAddress("001111e68297aae01347f6ce0ff21d5f72d3fa2a")

	msgs := make([]*types.StandardMessage, 0, 50000)

	ds_token_count := 16568
	start := 0
	for i := 0; i < ds_token_count; i++ {
		idx := start + i + 1
		native := ethTypes.NewMessage(from, &ds_address, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: ethCommon.RlpHash(&native),
		})
	}
	pk_count := 16638
	start = ds_token_count
	for i := 0; i < pk_count; i++ {
		idx := start + i + 1
		native := ethTypes.NewMessage(from, &kittyCore, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: ethCommon.RlpHash(&native),
		})
	}
	transfer_count := 16794
	start = ds_token_count + pk_count
	for i := 0; i < transfer_count; i++ {
		idx := start + i + 1
		native := ethTypes.NewMessage(from, &transfer, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: ethCommon.RlpHash(&native),
		})
	}
	scheduler, err := Start("history.binn")
	fmt.Printf(">>>>>>>>start log=%v\n", err)

	log := scheduler.Update([]*ethCommon.Address{&ds_address, &kittyCore}, []*ethCommon.Address{&ds_address, &kittyCore})
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
	addr1 := ethCommon.HexToAddress("fbc451fbd7e17a1e7b18347337657c1f2c52b631")
	addr2 := ethCommon.HexToAddress("2249977665260a63307cf72a4d65385cc0817cb5")
	addr3 := ethCommon.HexToAddress("1642dd5c38642f91e4aa0025978b572fe30ed89d")
	addr4 := ethCommon.HexToAddress("f5848bbe1a6ed5c4e9a5ba10c088bc28062a5d13")

	from := ethCommon.HexToAddress("001111e68297aae01347f6ce0ff21d5f72d3fa2a")

	msgs := make([]*types.StandardMessage, 0, 50000)

	ds_token_count := 0
	start := 0
	for i := 0; i < ds_token_count; i++ {
		idx := start + i + 1
		native := ethTypes.NewMessage(from, &addr1, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: ethCommon.RlpHash(&native),
		})
	}
	pk_count := 10
	start = ds_token_count
	for i := 0; i < pk_count; i++ {
		idx := start + i + 1
		native := ethTypes.NewMessage(from, &addr2, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: ethCommon.RlpHash(&native),
		})
	}
	transfer_count := 10
	start = ds_token_count + pk_count
	for i := 0; i < transfer_count; i++ {
		idx := start + i + 1
		native := ethTypes.NewMessage(from, &addr3, uint64(idx), big.NewInt(int64(idx)), uint64(idx*10), big.NewInt(int64(idx)), []byte{0, 1}, false)
		msgs = append(msgs, &types.StandardMessage{
			Source: 0,
			Native: &native,
			TxHash: ethCommon.RlpHash(&native),
		})
	}
	scheduler, err := Start("history.binn")
	fmt.Printf(">>>>>>>>start log=%v\n", err)

	log := scheduler.Update([]*ethCommon.Address{&addr1}, []*ethCommon.Address{&addr1})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log1=%v\n", log)
	}
	log = scheduler.Update([]*ethCommon.Address{&addr2}, []*ethCommon.Address{&addr2})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log2=%v\n", log)
	}
	log = scheduler.Update([]*ethCommon.Address{&addr3}, []*ethCommon.Address{&addr3})
	if len(log) != 0 {
		fmt.Printf(">>>>>>>>Update log3=%v\n", log)
	}
	log = scheduler.Update([]*ethCommon.Address{&addr4}, []*ethCommon.Address{&addr4})
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
