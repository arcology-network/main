package types

import (
	"crypto/sha256"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

type ConflictListItem struct {
	Left  *ethCommon.Address
	Right *ethCommon.Address
}

type ConflictList struct {
	list      []*ConflictListItem
	txaddress *TxAddressLookup
	txIds     *TxIdsLookup
	checker   map[[sha256.Size]byte]int
}

func NewConflictList(txaddress *TxAddressLookup, txIds *TxIdsLookup) *ConflictList {
	return &ConflictList{
		list:      []*ConflictListItem{},
		txaddress: txaddress,
		txIds:     txIds,
		checker:   map[[sha256.Size]byte]int{},
	}
}
func (cl *ConflictList) GetAddress(txid uint32) *ethCommon.Address {
	hash := cl.txIds.GetHash(txid)
	if hash == nil {
		return nil
	}
	return cl.txaddress.GetAddress(*hash)
}
func (cl *ConflictList) Add(cpairLeft, cpairRight []uint32) {
	if len(cpairLeft) == len(cpairRight) {
		for i := range cpairRight {
			left := cl.GetAddress(cpairLeft[i])
			right := cl.GetAddress(cpairRight[i])

			if left != nil && right != nil {
				//checker
				datas := make([]byte, 0, ethCommon.HashLength*2)
				datas = append(datas, left.Bytes()...)
				datas = append(datas, right.Bytes()...)
				hash := sha256.Sum256(datas)
				if _, ok := cl.checker[hash]; ok {
					continue
				}

				cl.list = append(cl.list, &ConflictListItem{
					Left:  left,
					Right: right,
				})
			}
		}
	}
}

func (cl *ConflictList) Clear() {
	cl.list = []*ConflictListItem{}
	cl.checker = map[[sha256.Size]byte]int{}
}

func (cl *ConflictList) GetAllPairs() []*ConflictListItem {
	return cl.list
}
