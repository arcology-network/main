package types

import (
	"sort"

	ethCommon "github.com/arcology-network/3rd-party/eth/common"
	"github.com/arcology-network/common-lib/common"
	urlcommon "github.com/arcology-network/concurrenturl/v2/common"
)

type SnapShotLookupItem struct {
	Size          int
	PrecedingHash ethCommon.Hash
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
	Snapshots map[ethCommon.Hash]*urlcommon.DatastoreInterface
}

func NewLookup() *SnapShotLookup {
	return &SnapShotLookup{
		Lookup:    Lookups{},
		Snapshots: map[ethCommon.Hash]*urlcommon.DatastoreInterface{},
	}
}

func (ss *SnapShotLookup) Reset(snapshop *urlcommon.DatastoreInterface) {
	nilHash := ethCommon.Hash{}
	ss.Lookup = Lookups{}
	ss.Snapshots = map[ethCommon.Hash]*urlcommon.DatastoreInterface{}

	ss.Lookup = append(ss.Lookup, SnapShotLookupItem{
		Size:          0,
		PrecedingHash: nilHash,
	})
	ss.Snapshots[nilHash] = snapshop
}

func (ss *SnapShotLookup) AddItem(precedingHash ethCommon.Hash, size int, snapshop *urlcommon.DatastoreInterface) {
	ss.Lookup = append(ss.Lookup, SnapShotLookupItem{
		Size:          size,
		PrecedingHash: precedingHash,
	})
	ss.Snapshots[precedingHash] = snapshop
	sort.Sort(ss.Lookup)
}

func (ss *SnapShotLookup) Query(precedings []*ethCommon.Hash) (*urlcommon.DatastoreInterface, []*ethCommon.Hash) {
	var sn *urlcommon.DatastoreInterface
	precedingSize := len(precedings)
	for _, item := range ss.Lookup {
		if item.Size > precedingSize {
			continue
		}
		if item.PrecedingHash == common.CalculateHash(precedings[:item.Size]) {
			return ss.Snapshots[item.PrecedingHash], precedings[item.Size:]
		}
	}

	return sn, precedings
}
