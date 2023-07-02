package backend

import (
	"fmt"
	"math/big"
	"testing"

	evmCommon "github.com/arcology-network/evm/common"
	"github.com/arcology-network/evm/core"
)

func TestMsgHash(t *testing.T) {
	msg := core.NewMessage(
		evmCommon.BytesToAddress([]byte{1}),
		nil,
		1,
		new(big.Int).SetUint64(1000),
		2000,
		new(big.Int).SetUint64(3000),
		[]byte{4},
		nil,
		false,
	)
	hash, _ := msgHash(&msg)
	t.Log(hash)
}
func TestID(t *testing.T) {
	fmt.Printf("%v", NewID())
}
