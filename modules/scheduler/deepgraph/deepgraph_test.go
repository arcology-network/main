package deepgraph

import (
	"reflect"
	"testing"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
)

// func TestSimpleInsertions(t *testing.T) {
// 	// 	   0  1
// 	// 	  /\ /
// 	//   2 1000
// 	//     / \
// 	//  1001  1002
// 	//     \  |
// 	//     2000
// 	//       3

// 	v0 := KeyType(ethCommon.BytesToHash([]byte("0")))
// 	v1 := KeyType(ethCommon.BytesToHash([]byte("1")))
// 	v2 := KeyType(ethCommon.BytesToHash([]byte("2")))
// 	v1000 := KeyType(ethCommon.BytesToHash([]byte("1000")))
// 	v1001 := KeyType(ethCommon.BytesToHash([]byte("1001")))
// 	v1002 := KeyType(ethCommon.BytesToHash([]byte("1002")))
// 	v2000 := KeyType(ethCommon.BytesToHash([]byte("2000")))
// 	v3 := KeyType(ethCommon.BytesToHash([]byte("3")))

// 	deepgraph := NewDeepgraph()

// 	deepgraph.Insert(
// 		[]KeyType{v0, v1},
// 		[]int{0, 1},
// 		[]VertexType{Original, Original},
// 		[][]KeyType{{}, {}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v2, v1000},
// 		[]int{2, 1000},
// 		[]VertexType{Original, Spawned},
// 		[][]KeyType{{v0}, {v0, v1}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v1001, v1002},
// 		[]int{1001, 1002},
// 		[]VertexType{Spawned, Spawned},
// 		[][]KeyType{{v1000}, {v1000}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v2000},
// 		[]int{2000},
// 		[]VertexType{Spawned},
// 		[][]KeyType{{v1001, v1002}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v3},
// 		[]int{3},
// 		[]VertexType{Original},
// 		[][]KeyType{{v2000}})
// 	deepgraph.IncreaseBatch()

// 	var desc [][]KeyType
// 	var ancs [][]KeyType

// 	desc, _, _ = deepgraph.GetDescendents([]KeyType{v2})
// 	if !reflect.DeepEqual(desc, [][]KeyType{{v2}}) {
// 		t.Error("v2: Wrong Decedents retrived !", desc)
// 	}

// 	desc, _, _ = deepgraph.GetAncestors([]KeyType{v2})
// 	if !reflect.DeepEqual(desc, [][]KeyType{{v0, v2}}) {
// 		t.Error("v2: Wrong Predecessors retrived !", desc)
// 	}

// 	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v1000})
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v1, v1000}}) {
// 		t.Error("v1000: Wrong Predecessors retrived !", ancs)
// 	}

// 	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v2000})
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v1, v1000, v1001, v1002, v2000}}) {
// 		t.Error("v2000: Wrong Predecessors retrived !", ancs)
// 	}

// 	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v3})
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v1, v1000, v1001, v1002, v2000, v3}}) {
// 		t.Error("v2000: Wrong Predecessors retrived !", ancs)
// 	}

// 	ancs, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v2})
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v2}}) {
// 		t.Error("v2: Wrong Predecessors retrived !", ancs)
// 	}

// 	ancs, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v2000})
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v1, v1000, v1001, v1002, v2000}}) {
// 		t.Error("v2000: Wrong Predecessors retrived !", ancs)
// 	}

// 	ancs, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v2000})
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v1, v1000, v1001, v1002, v2000}}) {
// 		t.Error("v2000: Wrong Predecessors retrived !", ancs)
// 	}

// 	desc, _, _ = deepgraph.GetDescendentFamilies([]KeyType{v1002})
// 	if !reflect.DeepEqual(desc, [][]KeyType{{v0, v1, v2, v1000, v1001, v1002, v2000, v3}}) {
// 		t.Error("v1002: Wrong Descendent Families retrived !", desc)
// 	}

// 	ancs, _, _ = deepgraph.GetSubgraphs()
// 	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v2}, {v0, v1, v1000, v1001, v1002, v2000, v3}}) {
// 		t.Error("GetSubgraphs: Wrong subgraphs retrived !", ancs)
// 	}
// }

// 	/* second graph */
// 	v7 := KeyType(ethCommon.BytesToHash([]byte("7")))
// 	v8 := KeyType(ethCommon.BytesToHash([]byte("8")))
// 	v9 := KeyType(ethCommon.BytesToHash([]byte("9")))

// 	deepgraphFrom := NewDeepgraph()

// 	deepgraph.Insert(
// 		[]KeyType{v7, v8},
// 		[]int{0, 1},
// 		[]VertexType{Original, Original},
// 		[][]KeyType{{}, {}})
// 	deepgraphFrom.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v9},
// 		[]int{2},
// 		[]VertexType{Spawned},
// 		[][]KeyType{{v7, v8}})
// 	deepgraphFrom.IncreaseBatch()

// 	if deepgraph.MoveFrom(deepgraphFrom); deepgraph.Size() != 11 || deepgraphFrom.Size() != 0 {
// 		t.Error("From deepgraph isn't empty !")
// 	}
// }

// func TestDeletion(t *testing.T) {
// 	// 	   0  1
// 	// 	  /\ /
// 	//   2 1000
// 	//     / \
// 	//  1001  1002
// 	//     \  |
// 	//     2000
// 	//       3

// 	v0 := KeyType(ethCommon.BytesToHash([]byte("0")))
// 	v1 := KeyType(ethCommon.BytesToHash([]byte("1")))
// 	v2 := KeyType(ethCommon.BytesToHash([]byte("2")))
// 	v1000 := KeyType(ethCommon.BytesToHash([]byte("1000")))
// 	v1001 := KeyType(ethCommon.BytesToHash([]byte("1001")))
// 	v1002 := KeyType(ethCommon.BytesToHash([]byte("1002")))
// 	v2000 := KeyType(ethCommon.BytesToHash([]byte("2000")))
// 	v3 := KeyType(ethCommon.BytesToHash([]byte("3")))

// 	deepgraph := NewDeepgraph()
// 	deepgraph.Insert(
// 		[]KeyType{v0, v1},
// 		[]int{0, 1},
// 		[]VertexType{Original, Original},
// 		[][]KeyType{{}, {}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v2, v1000},
// 		[]int{2, 1000},
// 		[]VertexType{Original, Spawned},
// 		[][]KeyType{{v0}, {v0, v1}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v1001, v1002},
// 		[]int{1001, 1002},
// 		[]VertexType{Spawned, Spawned},
// 		[][]KeyType{{v1000}, {v1000}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v2000},
// 		[]int{2000},
// 		[]VertexType{Spawned},
// 		[][]KeyType{{v1001, v1002}})
// 	deepgraph.IncreaseBatch()

// 	deepgraph.Insert(
// 		[]KeyType{v3},
// 		[]int{3},
// 		[]VertexType{Original},
// 		[][]KeyType{{v2000}})
// 	deepgraph.IncreaseBatch()

// 	desc, _, _ := deepgraph.GetSubgraphs()
// 	deepgraph.Remove(desc[0])

// 	desc, _, _ = deepgraph.GetSubgraphs()
// 	if !reflect.DeepEqual(desc, [][]KeyType{{v3}}) {
// 		t.Error("Remove !", desc)
// 	}
// }

func TestComplexInsertions(t *testing.T) {
	deepgraph := NewDeepgraph()

	/* vertices */
	v0 := KeyType(ethCommon.BytesToHash([]byte("0")))
	v1 := KeyType(ethCommon.BytesToHash([]byte("1")))
	v2 := KeyType(ethCommon.BytesToHash([]byte("2")))
	v3 := KeyType(ethCommon.BytesToHash([]byte("3")))
	v4 := KeyType(ethCommon.BytesToHash([]byte("4")))
	v9 := KeyType(ethCommon.BytesToHash([]byte("9")))

	v1000 := KeyType(ethCommon.BytesToHash([]byte("1000")))
	v1001 := KeyType(ethCommon.BytesToHash([]byte("1001")))
	v1002 := KeyType(ethCommon.BytesToHash([]byte("1002")))
	v1003 := KeyType(ethCommon.BytesToHash([]byte("1003")))
	v1004 := KeyType(ethCommon.BytesToHash([]byte("1004")))
	v1005 := KeyType(ethCommon.BytesToHash([]byte("1005")))
	v1006 := KeyType(ethCommon.BytesToHash([]byte("1006")))

	v5 := KeyType(ethCommon.BytesToHash([]byte("5")))
	v8 := KeyType(ethCommon.BytesToHash([]byte("8")))
	v10 := KeyType(ethCommon.BytesToHash([]byte("10")))
	v6 := KeyType(ethCommon.BytesToHash([]byte("6")))
	v11 := KeyType(ethCommon.BytesToHash([]byte("11")))
	v12 := KeyType(ethCommon.BytesToHash([]byte("12")))
	v1007 := KeyType(ethCommon.BytesToHash([]byte("1007")))
	v7 := KeyType(ethCommon.BytesToHash([]byte("7")))
	v13 := KeyType(ethCommon.BytesToHash([]byte("13")))
	v14 := KeyType(ethCommon.BytesToHash([]byte("14")))
	v15 := KeyType(ethCommon.BytesToHash([]byte("15")))
	v16 := KeyType(ethCommon.BytesToHash([]byte("16")))

	/*Batch 0*/
	deepgraph.Insert(
		[]KeyType{v0, v1, v2, v3, v4, v9},
		[]int{0, 1, 2, 3, 4, 9},
		[]VertexType{Original, Original, Original, Original, Original, Original, Original, Original, Original},
		[][]KeyType{{}, {}, {}, {}, {}, {}, {}, {}, {}})
	deepgraph.IncreaseBatch()

	/*Batch 1*/
	deepgraph.Insert(
		[]KeyType{v1000, v1001},
		[]int{1000, 1001},
		[]VertexType{Spawned, Spawned},
		[][]KeyType{{v1, v2}, {v3, v4}})
	deepgraph.IncreaseBatch()

	/*Batch 2*/
	deepgraph.Insert(
		[]KeyType{v1002},
		[]int{1002},
		[]VertexType{Spawned},
		[][]KeyType{{v1000, v1001}})
	deepgraph.IncreaseBatch()

	/*Batch 3*/
	deepgraph.Insert(
		[]KeyType{v1003, v1004},
		[]int{1003, 1004},
		[]VertexType{Spawned, Spawned},
		[][]KeyType{{v1002}, {v1002}})
	deepgraph.IncreaseBatch()

	/*Batch 5*/
	deepgraph.Insert(
		[]KeyType{v1005, v1006},
		[]int{1005, 1006},
		[]VertexType{Spawned, Spawned},
		[][]KeyType{{v1003}, {v1004}})
	deepgraph.IncreaseBatch()

	/*Batch 6*/
	deepgraph.Insert(
		[]KeyType{v5, v8, v10},
		[]int{5, 8, 10},
		[]VertexType{Original, Original, Original},
		[][]KeyType{{v0}, {v1}, {v9}})
	deepgraph.IncreaseBatch()

	/*Batch 7*/
	deepgraph.Insert(
		[]KeyType{v6, v11, v12, v15, v16},
		[]int{6, 11, 12, 15, 16},
		[]VertexType{Original, Original, Original, Original, Original},
		[][]KeyType{{v5}, {v8}, {v8}, {v10}, {v10}})
	deepgraph.IncreaseBatch()

	/*Batch 8*/
	deepgraph.Insert(
		[]KeyType{v1007},
		[]int{1007},
		[]VertexType{Spawned},
		[][]KeyType{{v6}})
	deepgraph.IncreaseBatch()

	/*Batch 9*/
	deepgraph.Insert(
		[]KeyType{v7},
		[]int{7},
		[]VertexType{Original},
		[][]KeyType{{v1007}})
	deepgraph.IncreaseBatch()

	/*Batch 10*/
	deepgraph.Insert(
		[]KeyType{v13},
		[]int{13},
		[]VertexType{Original},
		[][]KeyType{{v11, v12, v10}})
	deepgraph.IncreaseBatch()

	/*Batch 11*/
	deepgraph.Insert(
		[]KeyType{v14},
		[]int{14},
		[]VertexType{Original},
		[][]KeyType{{v3}})
	deepgraph.IncreaseBatch()

	/* Query */
	// desc, _, _ := deepgraph.GetDescendents([]KeyType{v0})
	// if !reflect.DeepEqual(desc, [][]KeyType{{v0, v5, v6, v1007, v7}}) {
	// 	t.Error("v0: Wrong Decedents retrived !", desc)
	// }
	// desc, _, _ = deepgraph.GetDescendents([]KeyType{v6})
	// if !reflect.DeepEqual(desc, [][]KeyType{{v6, v1007, v7}}) {
	// 	t.Error("v7: Wrong Decedents retrived !", desc)
	// }

	// desc, _, _ = deepgraph.GetDescendents([]KeyType{v1007})
	// if !reflect.DeepEqual(desc, [][]KeyType{{v1007, v7}}) {
	// 	t.Error("v1007: Wrong Decedents retrived !", desc)
	// }

	ancs, _, _ := deepgraph.GetAncestors([]KeyType{v6})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v0, v5, v6}}) {
		t.Error("v6: Wrong ancestors retrived !", ancs)
	}

	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v8})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v1, v8}}) {
		t.Error("v8: Wrong ancestors retrived !", ancs)
	}

	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v1002})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v1, v2, v3, v4, v1000, v1001, v1002}}) {
		t.Error("v1002: Wrong ancestors retrived !", ancs)
	}

	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v1000})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v1, v2, v1000}}) {
		t.Error("v1000: Wrong ancestors retrived !", ancs)
	}

	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v1001})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v3, v4, v1001}}) {
		t.Error("v1001: Wrong ancestors retrived !", ancs)
	}

	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v1002})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v1, v2, v3, v4, v1000, v1001, v1002}}) {
		t.Error("v1002: Wrong ancestors retrived !", ancs)
	}

	ancs, _, _ = deepgraph.GetAncestors([]KeyType{v1005})
	if !reflect.DeepEqual(ancs, [][]KeyType{{v1, v2, v3, v4, v1000, v1001, v1002, v1003, v1005}}) {
		t.Error("v1005: Wrong ancestors retrived !", ancs)
	}

	// 	desc, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v1001})
	// 	if !reflect.DeepEqual(desc, [][]KeyType{{v1007, v7}}) {
	// 		t.Error("v1007: Wrong Decedents retrived !", desc)
	// 	}
}

func TestDeepgraphDeletion(t *testing.T) {
	// 	   0  1  2  3  4   5
	// 	   \ /    \ | /   |
	//      6       7     8
	//     |              |
	//     9             10

	v0 := KeyType(ethCommon.BytesToHash([]byte("0")))
	v1 := KeyType(ethCommon.BytesToHash([]byte("1")))
	v2 := KeyType(ethCommon.BytesToHash([]byte("2")))
	v3 := KeyType(ethCommon.BytesToHash([]byte("3")))
	v4 := KeyType(ethCommon.BytesToHash([]byte("4")))
	v5 := KeyType(ethCommon.BytesToHash([]byte("5")))
	v6 := KeyType(ethCommon.BytesToHash([]byte("6")))
	v7 := KeyType(ethCommon.BytesToHash([]byte("7")))
	v8 := KeyType(ethCommon.BytesToHash([]byte("8")))
	v9 := KeyType(ethCommon.BytesToHash([]byte("9")))
	v10 := KeyType(ethCommon.BytesToHash([]byte("10")))

	deepgraph := NewDeepgraph()
	var hashes []KeyType

	hashes = []KeyType{v0, v1, v2, v3, v4, v5}
	deepgraph.Insert(hashes, deepgraph.Ids(hashes), []VertexType{Original, Original, Original, Original, Original, Original}, [][]KeyType{{}, {}, {}, {}, {}, {}})
	deepgraph.IncreaseBatch()

	hashes = []KeyType{v6, v7, v8}
	deepgraph.Insert([]KeyType{v6, v7, v8}, deepgraph.Ids(hashes), []VertexType{Spawned, Spawned, Spawned}, [][]KeyType{{v0, v1}, {v2, v3, v4}, {v5}})
	deepgraph.IncreaseBatch()

	hashes = []KeyType{v9, v10}
	deepgraph.Insert([]KeyType{v9, v10}, deepgraph.Ids(hashes), []VertexType{Original, Original}, [][]KeyType{{v6}, {v8}})
	deepgraph.IncreaseBatch()

	var desc [][]KeyType
	// desc, _, _ = deepgraph.GetDescendents([]KeyType{v0})
	// if !reflect.DeepEqual(desc, [][]KeyType{{v0, v6, v9}}) {
	// 	t.Error("v6: Wrong  Ancestors retrived !", desc)
	// }

	ance, _, _ := deepgraph.GetAncestors([]KeyType{v6})
	if !reflect.DeepEqual(ance, [][]KeyType{{v0, v1, v6}}) {
		t.Error("v6: Wrong  Ancestors retrived !", ance)
	}

	ance, _, _ = deepgraph.GetAncestors([]KeyType{v7})
	if !reflect.DeepEqual(ance, [][]KeyType{{v2, v3, v4, v7}}) {
		t.Error("v6: Wrong  Ancestors retrived !", ance)
	}

	ance, _, _ = deepgraph.GetAncestors([]KeyType{v9})
	if !reflect.DeepEqual(ance, [][]KeyType{{v0, v1, v6, v9}}) {
		t.Error("v9: Wrong Get Ancestors retrived !", ance)
	}

	ance, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v9})
	if !reflect.DeepEqual(ance, [][]KeyType{{v0, v1, v6, v9}}) {
		t.Error("v9: Wrong Get Ancestors retrived !", ance)
	}

	// ance, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v1})
	// if !reflect.DeepEqual(ance, [][]KeyType{{v0, v1, v6}}) {
	// 	t.Error("v9: Wrong Get Ancestors retrived !", ance)
	// }

	// ance, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v1})
	// if !reflect.DeepEqual(ance, [][]KeyType{{v0, v1, v6}}) {
	// 	t.Error("v9: Wrong Get Ancestors retrived !", ance)
	// }

	// ance, _, _ = deepgraph.GetAncestorFamilies([]KeyType{v1})
	// if !reflect.DeepEqual(ance, [][]KeyType{{v0, v1, v6}}) {
	// 	t.Error("v9: Wrong Get Ancestors retrived !", ance)
	// }

	// desc, _, _ = deepgraph.GetDescendentFamilies([]KeyType{v0})
	// if !reflect.DeepEqual(desc, [][]KeyType{{v0, v1, v6, v9}}) {
	// 	t.Error("v0: Wrong Descendent Families retrived !", desc)
	// }

	desc, _, _ = deepgraph.GetSubgraphs()
	if !reflect.DeepEqual(desc, [][]KeyType{{v2, v3, v4, v7}, {v0, v1, v6, v9}, {v5, v8, v10}}) {
		t.Error("GetSubgraphs: Wrong subgraphs retrived !", desc)
	}

	deepgraph.Remove(desc[0])
	desc, _, _ = deepgraph.GetSubgraphs()
	if !reflect.DeepEqual(desc, [][]KeyType{{v0, v1, v6, v9}, {v5, v8, v10}}) {
		t.Error("GetSubgraphs: Wrong subgraphs retrived !", desc)
	}
}
