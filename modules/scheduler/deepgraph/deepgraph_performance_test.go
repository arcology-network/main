package deepgraph

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

func PrepareData() []KeyType {
	stdMsgs := make([]ethCommon.Hash, 500000)
	for i := range stdMsgs {
		bytes := sha256.Sum256([]byte(fmt.Sprint(i)))
		stdMsgs[i] = (ethCommon.BytesToHash(bytes[0:32]))
	}

	return Keys{}.FromSource(stdMsgs)
}

func GetDataFields(hashes []KeyType) ([]KeyType, []int, []VertexType, [][]KeyType) {
	ids := make([]int, len(hashes))
	vTypes := make([]VertexType, len(hashes))
	predecessors := make([][]KeyType, len(hashes))
	for i := range hashes {
		ids[i] = i
		vTypes[i] = Original
		predecessors[i] = []KeyType{}
	}
	return hashes, ids, vTypes, predecessors
}

func TestDeepgraph(t *testing.T) {
	hashes, ids, vTypes, predecessors := GetDataFields(PrepareData())

	deepgraph := NewDeepgraph()

	t0 := time.Now()
	deepgraph.Insert(hashes, ids, vTypes, predecessors)
	fmt.Println("deepgroup.Size Originals():", deepgraph.Size())
	fmt.Println("deepgraph.Insert(): ", time.Now().Sub(t0))

	// keys := Keys.FromSource(hashes)
	t0 = time.Now()
	deepgraph.GetAncestorFamilies(hashes)
	fmt.Println("GetAncestorFamilies():", time.Now().Sub(t0))

	t0 = time.Now()
	deepgraph.GetSubgraphs()
	fmt.Println("deepgraph.GetSubgraphs(): ", time.Now().Sub(t0))

	t0 = time.Now()
	deepgraph.InsertEx(
		[]KeyType{KeyType(ethCommon.BytesToHash([]byte("-")))},
		[]int{99999},
		[]VertexType{Spawned},
		[][]KeyType{hashes})
	fmt.Println("deepgraph.Insert Spawned():", time.Now().Sub(t0))

	// t0 = time.Now()
	// deepgraph.GetAncestorFamilies([]KeyType{KeyType(ethCommon.BytesToHash([]byte("-")))})
	// fmt.Println("GetAncestorFamilies():", time.Now().Sub(t0))

	// t0 = time.Now()
	// deepgraph.GetSubgraphs()
	// fmt.Println("deepgraph.GetSubgraphs(): ", time.Now().Sub(t0))
}
