package types

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	EventSigner = common.BytesToHash(crypto.Keccak256([]byte("WriteBackEvent(bytes32,bytes32,bytes)")))
	receipts    = []*types.Receipt{

		&types.Receipt{
			Type:             1,
			TxHash:           common.HexToHash("0xbc3edd55a1e7862a0f184bb5334f10bfed2060b42ff90a61d9d9f18e7e2bcc27"),
			TransactionIndex: 0,
			Logs:             []*types.Log{},
		},
		&types.Receipt{
			Type:             1,
			TxHash:           common.HexToHash("0x20ac3469d577296923025eebbcabc5aa256689f3f620bb8d7ffa88ba195f00b3"),
			TransactionIndex: 1,
			Logs: []*types.Log{
				&types.Log{
					Address:     common.HexToAddress("0xB1e0e9e68297aAE01347F6Ce0ff21d5f72D3fa0F"),
					BlockNumber: 10,
					TxHash:      common.HexToHash("0x20ac3469d577296923025eebbcabc5aa256689f3f620bb8d7ffa88ba195f00b3"),
					TxIndex:     2,
					BlockHash:   common.HexToHash("0xfefafb3eaad615364302c2327a54743a4f9fe512e1148257436dd8920f12dc05"),
					Index:       0,
					Removed:     false,
					Data:        DecodeHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040"),
					Topics: []common.Hash{
						common.HexToHash("0xa6f1178b800eb43bf1caebba9cf7e34c21e955fffea903522fddd775ec5d0000"),
					},
				},
			},
		},
		&types.Receipt{
			Type:             1,
			TxHash:           common.HexToHash("0x2563db68b15ac0c72be186ceae46f62133e3fef6cb2ff04db54480902398ec4d"),
			TransactionIndex: 2,
			Logs: []*types.Log{
				&types.Log{
					Address:     common.HexToAddress("0xB1e0e9e68297aAE01347F6Ce0ff21d5f72D3fa0F"),
					BlockNumber: 10,
					TxHash:      common.HexToHash("0x2563db68b15ac0c72be186ceae46f62133e3fef6cb2ff04db54480902398ec4d"),
					TxIndex:     2,
					BlockHash:   common.HexToHash("0xfefafb3eaad615364302c2327a54743a4f9fe512e1148257436dd8920f12dc05"),
					Index:       0,
					Removed:     false,
					Data:        DecodeHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040000000000000000000000000b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f0000000000000000000000000000000000000000000000000000000000000003"),
					Topics: []common.Hash{
						common.HexToHash("0xa6f1178b800eb43bf1caebba9cf7e34c21e955fffea903522fddd775ec5d8c64"),
						common.HexToHash("0x2563db68b15ac0c72be186ceae46f62133e3fef6cb2ff04db54480902398ec4d"),
						common.HexToHash("0xf7c779192e208dfbaaf6c15255469eb06b751734b3ca18c5a1d01e89e8ead25d"),
					},
				},
				&types.Log{
					Address:     common.HexToAddress("0xB1e0e9e68297aAE01347F6Ce0ff21d5f72D3fa0F"),
					BlockNumber: 10,
					TxHash:      common.HexToHash("0x20ac3469d577296923025eebbcabc5aa256689f3f620bb8d7ffa88ba195f00b3"),
					TxIndex:     2,
					BlockHash:   common.HexToHash("0xfefafb3eaad615364302c2327a54743a4f9fe512e1148257436dd8920f12dc05"),
					Index:       1,
					Removed:     false,
					Data:        DecodeHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040000000000000000000000000b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f0000000000000000000000000000000000000000000000000000000000000002"),
					Topics: []common.Hash{
						common.HexToHash("0xa6f1178b800eb43bf1caebba9cf7e34c21e955fffea903522fddd775ec5d8c64"),
						common.HexToHash("0x20ac3469d577296923025eebbcabc5aa256689f3f620bb8d7ffa88ba195f00b3"),
						common.HexToHash("0xf7c779192e208dfbaaf6c15255469eb06b751734b3ca18c5a1d01e89e8ead25d"),
					},
				},
				&types.Log{
					Address:     common.HexToAddress("0xB1e0e9e68297aAE01347F6Ce0ff21d5f72D3fa0F"),
					BlockNumber: 10,
					TxHash:      common.HexToHash("0xbc3edd55a1e7862a0f184bb5334f10bfed2060b42ff90a61d9d9f18e7e2bcc27"),
					TxIndex:     2,
					BlockHash:   common.HexToHash("0xfefafb3eaad615364302c2327a54743a4f9fe512e1148257436dd8920f12dc05"),
					Index:       2,
					Removed:     false,
					Data:        DecodeHex("00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000040000000000000000000000000b1e0e9e68297aae01347f6ce0ff21d5f72d3fa0f0000000000000000000000000000000000000000000000000000000000000001"),
					Topics: []common.Hash{
						common.HexToHash("0xa6f1178b800eb43bf1caebba9cf7e34c21e955fffea903522fddd775ec5d8c64"),
						common.HexToHash("0xbc3edd55a1e7862a0f184bb5334f10bfed2060b42ff90a61d9d9f18e7e2bcc27"),
						common.HexToHash("0xf7c779192e208dfbaaf6c15255469eb06b751734b3ca18c5a1d01e89e8ead25d"),
					},
				},
			},
		},
	}
)

func DecodeHex(hexstr string) []byte {
	b, _ := hex.DecodeString(hexstr)
	return b
}

func TestHash(t *testing.T) {
	fmt.Printf("EventSigner:%x\n", EventSigner)
	destReceipts := PostProcess(receipts)
	fmt.Printf("destReceipts:%v\n", destReceipts)
}

func PostProcess(receipts []*types.Receipt) []*types.Receipt {
	logs := make([]*types.Log, 0, 50000)
	mpReceipts := make(map[common.Hash]*types.Receipt, len(receipts))
	//filter defer logs
	for i := range receipts {
		tmpLogs := make([]*types.Log, 0, len(receipts[i].Logs))
		for j := range receipts[i].Logs {
			if receipts[i].Logs[j].Topics[0] == EventSigner {
				//collect defer logs
				logs = append(logs, receipts[i].Logs[j])
			} else {
				//remove current log from src
				receipts[i].Logs[j].Index = uint(len(tmpLogs))
				tmpLogs = append(tmpLogs, receipts[i].Logs[j])
			}
		}
		receipts[i].Logs = tmpLogs
		mpReceipts[receipts[i].TxHash] = receipts[i]
	}

	//update ldefer log into parallel transactions
	for i := range logs {
		txhash := logs[i].Topics[1]
		destReceipt := mpReceipts[txhash]
		logs[i].TxHash = txhash
		logs[i].TxIndex = destReceipt.TransactionIndex
		logs[i].Index = uint(len(destReceipt.Logs))
		logs[i].Topics = []common.Hash{logs[i].Topics[2]}
		logs[i].Data = logs[i].Data[64:]
		destReceipt.Logs = append(destReceipt.Logs, logs[i])
		mpReceipts[txhash] = destReceipt
	}

	//setup dest receipts
	destReceipts := make([]*types.Receipt, 0, len(mpReceipts))
	for i := range receipts {
		destReceipts = append(destReceipts, mpReceipts[receipts[i].TxHash])
	}

	return destReceipts
}
