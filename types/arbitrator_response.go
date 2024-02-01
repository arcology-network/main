package types

import ethCommon "github.com/ethereum/go-ethereum/common"

type ArbitratorResponse struct {
	ConflictedList []*ethCommon.Hash
	CPairLeft      []uint32
	CPairRight     []uint32
}
