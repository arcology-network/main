package types

import (
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type JobSequenceResponse struct {
	Responses       []*ExecuteResponse
	ContractAddress []evmCommon.Address
	CallResults     [][]byte
}

type ExecResponses struct {
	Resp   []*JobSequenceResponse
	ExecId uint32
}
