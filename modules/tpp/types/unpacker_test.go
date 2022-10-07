package types

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethRlp "github.com/HPISTechnologies/3rd-party/eth/rlp"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
)

func TestRlp(t *testing.T) {
	txnums := 15000
	txs := make([]*ethTypes.Transaction, 0, txnums)

	for i := range txs {
		txs[i] = ethTypes.NewTransactionTS(
			0,
			uint64(i),
			ethCommon.BigToAddress(big.NewInt(int64(i))),
			ethCommon.BigToAddress(big.NewInt(int64(i))),
			big.NewInt(int64(i)),
			uint64(i),
			big.NewInt(int64(i)),
			[]byte{byte(i % 255), byte((i + 1) % 255), byte((i + 2) % 255)},
		)
	}
	txsRawData := make([][]byte, 0, txnums)
	startTime := time.Now()
	for i := range txs {
		data, err := ethRlp.EncodeToBytes(txs[i])
		if err != nil {
			fmt.Printf("eocode txs[%d] err: %v!\n", i, err)
			return
		}
		txsRawData[i] = data
	}
	fmt.Printf("encode txsnums=%v transactions time=%v ms\n", txnums, time.Now().Sub(startTime))
	startTime1 := time.Now()
	for i := range txsRawData {
		otx := new(ethTypes.Transaction)
		if err := ethRlp.DecodeBytes(txsRawData[i], otx); err != nil {
			fmt.Printf("deocode txs[%d] err: %v!\n", i, err)
			return
		}
		ethCommon.RlpHash(otx)

	}
	fmt.Printf("dencode txsnums=%v transactions time=%v ms\n", txnums, time.Now().Sub(startTime1))
}
