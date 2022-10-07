package types

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
)

type ContractLookup struct {
	db map[ethCommon.Address]int
}

func NewContractLookup() *ContractLookup {
	lookup := &ContractLookup{
		db: map[ethCommon.Address]int{},
	}
	return lookup
}

func (lookup *ContractLookup) UpdateContract(addrs []ethCommon.Address) {
	for _, addr := range addrs {
		lookup.db[addr] = 1
	}
}
func (lookup *ContractLookup) GetAll() map[ethCommon.Address]int {
	return lookup.db
}

func (lookup *ContractLookup) Divide(stdMsgs []*types.StandardMessage) ([]*types.StandardMessage, []*types.StandardMessage) {
	contracts := make([]*types.StandardMessage, 0, len(stdMsgs))
	transfers := make([]*types.StandardMessage, 0, len(stdMsgs))

	for i := range stdMsgs {
		to := stdMsgs[i].Native.To()
		if to == nil {
			transfers = append(transfers, stdMsgs[i])
			continue
		}
		if _, ok := lookup.db[*to]; ok {
			contracts = append(contracts, stdMsgs[i])
		} else {
			transfers = append(transfers, stdMsgs[i])
		}
	}
	return transfers, contracts
}
