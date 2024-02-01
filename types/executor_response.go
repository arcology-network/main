package types

import (
	codec "github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/types"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type ExecuteResponse struct {
	Hash    ethCommon.Hash
	Status  uint64
	GasUsed uint64
}

type ExecutorResponses struct {
	HashList    []ethCommon.Hash
	StatusList  []uint64
	GasUsedList []uint64

	ContractAddresses []ethCommon.Address

	CallResults [][]byte
}

func (er *ExecutorResponses) GobEncode() ([]byte, error) {
	data := [][]byte{
		types.Hashes(er.HashList).Encode(),
		codec.Uint64s(er.StatusList).Encode(),
		codec.Uint64s(er.GasUsedList).Encode(),

		types.Addresses(er.ContractAddresses).Encode(),

		codec.Byteset(er.CallResults).Encode(),
	}
	return codec.Byteset(data).Encode(), nil
}
func (er *ExecutorResponses) GobDecode(data []byte) error {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	er.HashList = types.Hashes(er.HashList).Decode(fields[0])
	er.StatusList = []uint64(codec.Uint64s(er.StatusList).Decode(fields[1]).(codec.Uint64s))
	er.GasUsedList = []uint64(codec.Uint64s(er.GasUsedList).Decode(fields[2]).(codec.Uint64s))

	er.ContractAddresses = types.Addresses(er.ContractAddresses).Decode(fields[3])

	er.CallResults = [][]byte(codec.Byteset{}.Decode(fields[4]).(codec.Byteset))
	return nil
}
