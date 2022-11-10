package backend

import (
	"fmt"
	"math/big"
	"testing"

	thdcmn "github.com/arcology-network/3rd-party/eth/common"
	thdtyp "github.com/arcology-network/3rd-party/eth/types"
)

func TestMsgHash(t *testing.T) {
	msg := thdtyp.NewMessage(
		thdcmn.BytesToAddress([]byte{1}),
		nil,
		1,
		new(big.Int).SetUint64(1000),
		2000,
		new(big.Int).SetUint64(3000),
		[]byte{4},
		false,
	)
	hash, _ := msgHash(&msg)
	t.Log(hash)
}
func TestID(t *testing.T) {
	fmt.Printf("%v", NewID())
}
