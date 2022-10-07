package types

import (
	"sync"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

type CurrencyArray struct {
	deleteTxs []*ethCommon.Hash
	lock      sync.RWMutex
}

func NewCurrencyArray() *CurrencyArray {
	return &CurrencyArray{
		deleteTxs: []*ethCommon.Hash{},
	}
}
func (ca *CurrencyArray) Append(hash *ethCommon.Hash) {
	ca.lock.Lock()
	defer ca.lock.Unlock()

	ca.deleteTxs = append(ca.deleteTxs, hash)
}

func (ca *CurrencyArray) GetAll() []*ethCommon.Hash {
	ca.lock.Lock()
	defer ca.lock.Unlock()
	return ca.deleteTxs
}
