package storage

import (
	"testing"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
)

func TestWriteRead(t *testing.T) {
	store := NewSchdStore(1, "tester").(*SchdStore)
	store.Config(map[string]interface{}{
		"root": "./schdstore/",
	})

	store.writeToFile(&SchdState{
		Height:            1,
		NewContracts:      []ethcmn.Address{ethcmn.BytesToAddress([]byte{1}), ethcmn.BytesToAddress([]byte{2})},
		ConflictionLefts:  []ethcmn.Address{ethcmn.BytesToAddress([]byte{3}), ethcmn.BytesToAddress([]byte{4})},
		ConflictionRights: []ethcmn.Address{ethcmn.BytesToAddress([]byte{5}), ethcmn.BytesToAddress([]byte{6})},
	})
	store.writeToFile(&SchdState{
		Height:            2,
		NewContracts:      []ethcmn.Address{ethcmn.BytesToAddress([]byte{7}), ethcmn.BytesToAddress([]byte{8})},
		ConflictionLefts:  []ethcmn.Address{},
		ConflictionRights: []ethcmn.Address{},
	})
	store.writeToFile(&SchdState{
		Height:            3,
		NewContracts:      []ethcmn.Address{},
		ConflictionLefts:  []ethcmn.Address{ethcmn.BytesToAddress([]byte{9}), ethcmn.BytesToAddress([]byte{10})},
		ConflictionRights: []ethcmn.Address{ethcmn.BytesToAddress([]byte{11}), ethcmn.BytesToAddress([]byte{12})},
	})

	var states []SchdState
	store.readFromFile(&states)
	t.Log(states)
}
