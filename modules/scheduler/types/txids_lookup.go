package types

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
)

type TxIdsLookup struct {
	ids       map[ethCommon.Hash]uint32
	idsRevert map[uint32]ethCommon.Hash
}

func NewTxIdsLookup() *TxIdsLookup {
	return &TxIdsLookup{
		ids:       map[ethCommon.Hash]uint32{},
		idsRevert: map[uint32]ethCommon.Hash{},
	}
}

func (tl *TxIdsLookup) Add(txHash ethCommon.Hash, txid uint32) {
	tl.ids[txHash] = txid
	tl.idsRevert[txid] = txHash
}

func (tl *TxIdsLookup) Clear() {
	tl.ids = map[ethCommon.Hash]uint32{}
	tl.idsRevert = map[uint32]ethCommon.Hash{}
}

func (tl *TxIdsLookup) GetId(txHash ethCommon.Hash) uint32 {
	if id, ok := tl.ids[txHash]; ok {
		return id
	} else {
		return 0
	}
}
func (tl *TxIdsLookup) GetHash(txid uint32) *ethCommon.Hash {
	if hash, ok := tl.idsRevert[txid]; ok {
		return &hash
	} else {
		return nil
	}
}

func (tl *TxIdsLookup) Adds(sequences []*types.ExecutingSequence) {
	for _, sequence := range sequences {
		for j := range sequence.Msgs {
			tl.Add(sequence.Msgs[j].TxHash, sequence.Txids[j])
		}
	}
}
func (tl *TxIdsLookup) Append(txids map[ethCommon.Hash]uint32) {
	for hash, id := range txids {
		tl.Add(hash, id)
	}
}

func InitTxIds(sequence *types.ExecutingSequence, start uint32) uint32 {
	for i := range sequence.Msgs {
		id := start << 8
		sequence.Txids[i] = id
		start = start + 1
	}
	return start
}

func CreateIds(start uint32, sizes int) ([]uint32, uint32) {
	ids := make([]uint32, sizes)
	for i := 0; i < sizes; i++ {
		id := start << 8
		ids[i] = id
		start = start + 1
	}
	return ids, start
}
func GetSpawnedTxIds(ids []uint32) []uint32 {
	for i := range ids {
		ids[i] = ids[i] + 1
	}
	return ids
}
