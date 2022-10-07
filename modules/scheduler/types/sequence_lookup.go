package types

import (
	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

type SequenceLookup struct {
	zoomIn  map[ethCommon.Hash][]ethCommon.Hash
	zoomOut map[ethCommon.Hash]ethCommon.Hash
}

func NewSequenceLookup() *SequenceLookup {
	return &SequenceLookup{
		zoomIn:  map[ethCommon.Hash][]ethCommon.Hash{},
		zoomOut: map[ethCommon.Hash]ethCommon.Hash{},
	}
}

func (sl *SequenceLookup) Add(sequenceId ethCommon.Hash, list []ethCommon.Hash) {
	sl.zoomIn[sequenceId] = list
	for _, hash := range list {
		sl.zoomOut[hash] = sequenceId
	}
}
func (sl *SequenceLookup) Clear() {
	sl.zoomIn = map[ethCommon.Hash][]ethCommon.Hash{}
	sl.zoomOut = map[ethCommon.Hash]ethCommon.Hash{}
}

func (sl *SequenceLookup) Zoomin(list []*ethCommon.Hash) []*ethCommon.Hash {
	out := make([]*ethCommon.Hash, 0, len(list))
	for _, hash := range list {
		if zlist, ok := sl.zoomIn[*hash]; ok {
			for _, iHash := range zlist {
				outHash := iHash
				out = append(out, &outHash)
			}
		}
	}
	return out
}

func (sl *SequenceLookup) Zoomout(hash ethCommon.Hash) *ethCommon.Hash {
	if retHash, ok := sl.zoomOut[hash]; ok {
		return &retHash
	}
	return nil
}
