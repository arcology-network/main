package types

import (
	"sort"

	"github.com/arcology-network/concurrenturl/interfaces"
	"github.com/arcology-network/main/modules/tools"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type SnapShotLookupItem struct {
	Size          int
	PrecedingHash evmCommon.Hash
}

type Lookups []SnapShotLookupItem

// Len()
func (l Lookups) Len() int {
	return len(l)
}

// Less():
func (l Lookups) Less(i, j int) bool {
	return l[i].Size > l[j].Size
}

// Swap()
func (l Lookups) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

type SnapShotLookup struct {
	Lookup    Lookups
	Snapshots map[evmCommon.Hash]*interfaces.Datastore
}

func NewLookup() *SnapShotLookup {
	return &SnapShotLookup{
		Lookup:    Lookups{},
		Snapshots: map[evmCommon.Hash]*interfaces.Datastore{},
	}
}

func (ss *SnapShotLookup) Reset(snapshop *interfaces.Datastore) {
	nilHash := evmCommon.Hash{}
	ss.Lookup = Lookups{}
	ss.Snapshots = map[evmCommon.Hash]*interfaces.Datastore{}

	ss.Lookup = append(ss.Lookup, SnapShotLookupItem{
		Size:          0,
		PrecedingHash: nilHash,
	})
	ss.Snapshots[nilHash] = snapshop
}

func (ss *SnapShotLookup) AddItem(precedingHash evmCommon.Hash, size int, snapshop *interfaces.Datastore) {
	ss.Lookup = append(ss.Lookup, SnapShotLookupItem{
		Size:          size,
		PrecedingHash: precedingHash,
	})
	ss.Snapshots[precedingHash] = snapshop
	sort.Sort(ss.Lookup)
}

func (ss *SnapShotLookup) Query(precedings []*evmCommon.Hash) (*interfaces.Datastore, []*evmCommon.Hash) {
	var sn *interfaces.Datastore
	precedingSize := len(precedings)
	for _, item := range ss.Lookup {
		if item.Size > precedingSize {
			continue
		}
		if item.PrecedingHash == tools.CalculateHash(precedings[:item.Size]) {
			return ss.Snapshots[item.PrecedingHash], precedings[item.Size:]
		}
	}

	return sn, precedings
}
