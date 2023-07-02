package storage

import (
	"testing"

	evmCommon "github.com/arcology-network/evm/common"
)

func TestWriteRead(t *testing.T) {
	store := NewSchdStore(1, "tester").(*SchdStore)
	store.Config(map[string]interface{}{
		"root": "./schdstore/",
	})

	store.writeToFile(&SchdState{
		Height:            1,
		NewContracts:      []evmCommon.Address{evmCommon.BytesToAddress([]byte{1}), evmCommon.BytesToAddress([]byte{2})},
		ConflictionLefts:  []evmCommon.Address{evmCommon.BytesToAddress([]byte{3}), evmCommon.BytesToAddress([]byte{4})},
		ConflictionRights: []evmCommon.Address{evmCommon.BytesToAddress([]byte{5}), evmCommon.BytesToAddress([]byte{6})},
	})
	store.writeToFile(&SchdState{
		Height:            2,
		NewContracts:      []evmCommon.Address{evmCommon.BytesToAddress([]byte{7}), evmCommon.BytesToAddress([]byte{8})},
		ConflictionLefts:  []evmCommon.Address{},
		ConflictionRights: []evmCommon.Address{},
	})
	store.writeToFile(&SchdState{
		Height:            3,
		NewContracts:      []evmCommon.Address{},
		ConflictionLefts:  []evmCommon.Address{evmCommon.BytesToAddress([]byte{9}), evmCommon.BytesToAddress([]byte{10})},
		ConflictionRights: []evmCommon.Address{evmCommon.BytesToAddress([]byte{11}), evmCommon.BytesToAddress([]byte{12})},
	})

	var states []SchdState
	store.readFromFile(&states)
	t.Log(states)
}
