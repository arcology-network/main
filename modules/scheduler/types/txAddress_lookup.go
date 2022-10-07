package types

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
)

type TxAddressLookup struct {
	dbs map[ethCommon.Hash]ethCommon.Address
}

func NewTxAddressLookup() *TxAddressLookup {
	return &TxAddressLookup{
		dbs: map[ethCommon.Hash]ethCommon.Address{},
	}
}

func (tl *TxAddressLookup) Add(txHash ethCommon.Hash, addr ethCommon.Address) {
	tl.dbs[txHash] = addr
}

func (tl *TxAddressLookup) Clear() {
	tl.dbs = map[ethCommon.Hash]ethCommon.Address{}
}

func (tl *TxAddressLookup) GetAddress(txHash ethCommon.Hash) *ethCommon.Address {
	if addr, ok := tl.dbs[txHash]; ok {
		return &addr
	} else {
		return nil
	}
}

func (tl *TxAddressLookup) GeContractAddress(txHash ethCommon.Hash) ethCommon.Address {
	if addr, ok := tl.dbs[txHash]; ok {
		return addr
	} else {
		return ethCommon.Address{}
	}
}
func (tl *TxAddressLookup) Reset(messages []*types.StandardMessage) {
	tl.dbs = map[ethCommon.Hash]ethCommon.Address{}
	for _, msg := range messages {
		if msg.Native.To() != nil {
			tl.dbs[msg.TxHash] = *msg.Native.To()
		}
	}
}
func (tl *TxAddressLookup) Append(msgs map[ethCommon.Hash]ethCommon.Address) {
	for h, addr := range msgs {
		tl.dbs[h] = addr
	}
}
