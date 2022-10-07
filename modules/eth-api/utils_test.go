package ethapi

import (
	"encoding/hex"
	"math/big"
	"testing"
)

func TestHexEncodeToString(t *testing.T) {
	str := hex.EncodeToString([]byte{0, 0, 0})
	t.Log(str)
}

func TestNumberToHex(t *testing.T) {
	t.Log(NumberToHex(new(big.Int).SetUint64(100)))
	t.Log(NumberToHex(uint64(100)))
}
