package types

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	ethTypes "github.com/arcology-network/3rd-party/eth/types"
	"github.com/arcology-network/common-lib/types"
)

func TestQueue(t *testing.T) {
	contractAddr1 := ethCommon.BytesToAddress([]byte{1, 2, 3})
	contractAddr2 := ethCommon.BytesToAddress([]byte{4, 5, 6})
	contractAddr3 := ethCommon.BytesToAddress([]byte{7, 8, 9})
	data := []byte{11, 12, 13}
	message1 := ethTypes.NewMessage(contractAddr1, &contractAddr1, 1, new(big.Int).SetInt64(1), 1e9, new(big.Int).SetInt64(1), data, false)
	standardMessager1 := types.StandardMessage{
		Native: &message1,
		TxHash: ethCommon.RlpHash(message1),
	}
	message2 := ethTypes.NewMessage(contractAddr2, &contractAddr2, 2, new(big.Int).SetInt64(2), 1e9, new(big.Int).SetInt64(2), data, false)
	standardMessager2 := types.StandardMessage{
		Native: &message2,
		TxHash: ethCommon.RlpHash(message2),
	}
	message3 := ethTypes.NewMessage(contractAddr3, &contractAddr3, 3, new(big.Int).SetInt64(3), 1e9, new(big.Int).SetInt64(3), data, false)
	standardMessager3 := types.StandardMessage{
		Native: &message3,
		TxHash: ethCommon.RlpHash(message3),
	}
	msgs := []*types.StandardMessage{&standardMessager1, &standardMessager2}
	ids := []uint32{1, 2}
	q := NewQueue()
	q.Reset(msgs, ids)

	fmt.Printf("idx=%v,txs=%v\n", q.idx, q.txs)

	m1, _ := q.GetNext()
	fmt.Printf("idx=%v,txs=%v\n", q.idx, q.txs)
	if !reflect.DeepEqual(*m1, standardMessager1) {
		t.Error("GetNext err!", *m1, standardMessager1)
	}

	q.Insert(&standardMessager3, uint32(3))

	fmt.Printf("idx=%v,txs=%v\n", q.idx, q.txs)

	m3, _ := q.GetNext()
	if !reflect.DeepEqual(*m3, standardMessager3) {
		t.Error("Insert err!", *m3, standardMessager3)
	}
}
