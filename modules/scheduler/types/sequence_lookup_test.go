package types

import (
	"fmt"
	"reflect"
	"testing"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

func TestSequenceLookup(t *testing.T) {
	lookup := NewSequenceLookup()

	hash1 := ethCommon.BytesToHash([]byte{1})
	hash2 := ethCommon.BytesToHash([]byte{2})
	hash3 := ethCommon.BytesToHash([]byte{3})
	hash4 := ethCommon.BytesToHash([]byte{4})
	hash5 := ethCommon.BytesToHash([]byte{5})
	hash6 := ethCommon.BytesToHash([]byte{6})

	lookup.Add(hash1, []ethCommon.Hash{hash2, hash3})
	lookup.Add(hash4, []ethCommon.Hash{hash5, hash6})

	list := lookup.Zoomin([]*ethCommon.Hash{&hash1})
	if !reflect.DeepEqual(list, []*ethCommon.Hash{&hash2, &hash3}) {
		t.Error("ZoomIn err!", list, []*ethCommon.Hash{&hash2, &hash3})
	}
	fmt.Printf("list=%v\n", list)

	sequenceId := lookup.Zoomout(hash5)
	if !reflect.DeepEqual(sequenceId, &hash4) {
		t.Error("ZoomOut err!", sequenceId, &hash4)
	}
	fmt.Printf("sequenceId=%v\n", sequenceId)

	lookup.Clear()
	if !reflect.DeepEqual(len(lookup.zoomIn), 0) {
		t.Error("Clear err!", len(lookup.zoomIn), 0)
	}
	if !reflect.DeepEqual(len(lookup.zoomOut), 0) {
		t.Error("Clear err!", len(lookup.zoomOut), 0)
	}
	fmt.Printf("zoomin=%v\n", lookup.zoomIn)
	fmt.Printf("zoomout=%v\n", lookup.zoomOut)
}
