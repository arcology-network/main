package deepgraph

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/common"
)

type KeyType ethCommon.Hash
type VertexType int

const (
	Unknown  VertexType = 0
	Original VertexType = 1
	Spawned  VertexType = 2
)

type Keys []KeyType

func (_ Keys) FromSource(source []ethCommon.Hash) []KeyType {
	keys := make([]KeyType, len(source))
	for i, v := range source {
		keys[i] = KeyType(v)
	}
	return keys
}

type Keyset [][]KeyType

func (hashes Keyset) flatten() []KeyType {
	buffer := make([]KeyType, len(hashes))
	for i := 0; i < len(hashes); i++ {
		buffer = append(buffer, hashes[i]...)
	}
	return buffer
}

type Keygroup [][][]KeyType

func (hashgroup Keygroup) flatten() [][]KeyType {
	buffer := make([][]KeyType, len(hashgroup))
	for i := 0; i < len(hashgroup); i++ {
		buffer = append(buffer, hashgroup[i]...)
	}
	return buffer
}

type Vertex struct {
	msg    KeyType // tx hash
	id     int     // vertex id
	family int     // family id, same as the id initially
	batch  int
	vType  VertexType
	// parents  []*Vertex // parents
	// child []*Vertex

	parents []*Vertex // parents
	child   *Vertex

	cachedAncestors    []*Vertex //all the parents
	cachedAnceKeys     []KeyType
	cachedAnceBatches  []int
	cachedAnceFamilies []int
	mtx                sync.Mutex
}

type Direction func(from *Vertex, buffer []*Vertex) []*Vertex
type Action func(from *Vertex, direction Direction, buffer []*Vertex) []*Vertex

// link vertices vertex to its Predecessors
func (vertex *Vertex) Link(parents []*Vertex) {
	vertex.parents = parents
	for i := 0; i < len(parents); i++ {
		parents[i].child = vertex
	}
}

func (vertex *Vertex) traverse(buffer []*Vertex, direction Direction, action Action) {
	if vertex != nil {
		for _, v := range action(vertex, direction, buffer) {
			v.traverse(buffer, direction, action)
		}
	}
}

func (vertex *Vertex) CacheAncestors() {
	if len(vertex.parents) == 0 {
		return
	}

	parents := make([]*Vertex, 0, len(vertex.parents))
	for _, parent := range vertex.parents {
		parents = append(parents, parent)
	}

	buffer := make([][]*Vertex, len(parents))
	appender := func(start, end, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			buffer[i] = append(parents[i].cachedAncestors, parents[i])
		}
	}

	common.ParallelWorker(len(buffer), 64, appender)

	vertex.cachedAncestors = QuickSort(append(vertex.cachedAncestors, Vertexset(buffer).flatten()...))
	vertex.cachedAnceKeys, vertex.cachedAnceBatches, vertex.cachedAnceFamilies = Vertices{}.Export(vertex.cachedAncestors)
}

func (vertex *Vertex) Print() {
	fmt.Println("V: ", vertex.id, " <= ", vertex.family)
	fmt.Print("Parents: ")
	for _, pred := range vertex.parents {
		fmt.Print(pred.id)
		fmt.Print(", ")
	}
	fmt.Println()

	fmt.Print("Child: ")
	fmt.Print(vertex.child.id)
	fmt.Println()
}

func VertexFromSource(hash KeyType, id int, batch int, vType VertexType) *Vertex {
	return &Vertex{
		msg:     hash,
		id:      id,
		family:  id,
		batch:   batch,
		vType:   vType,
		parents: []*Vertex{},
		child:   nil,
	}
}

func VerticesFromSource(hashes []KeyType, ids []int, batch int, vTypes []VertexType) *[]*Vertex {
	vertices := make([]*Vertex, len(hashes))
	for i, v := range hashes {
		vertices[i] = VertexFromSource(v, ids[i], batch, vTypes[i])
	}
	return &vertices
}

type Vertices []*Vertex

func (v Vertices) Len() int      { return len(v) }
func (v Vertices) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v Vertices) Less(i, j int) bool {
	if v[i].batch == v[j].batch {
		return v[i].id < v[j].id
	}
	return v[i].batch < v[j].batch
}

func (_ Vertices) Export(vtx []*Vertex) ([]KeyType, []int, []int) {
	return Vertices(vtx).keys(), Vertices(vtx).batches(), Vertices(vtx).families()
}

func (v Vertices) keys() []KeyType {
	buffer := make([]KeyType, 0, len(v)+1)
	for i := range v {
		buffer = append(buffer, v[i].msg)
	}
	return buffer
}

func (v Vertices) ids() []int {
	buffer := make([]int, 0, len(v)+1)
	for i := range v {
		buffer = append(buffer, v[i].id)
	}
	return buffer
}

func (v Vertices) batches() []int {
	buffer := make([]int, 0, len(v)+1)
	for i := range v {
		buffer = append(buffer, v[i].batch)
	}
	return buffer
}

func (v Vertices) vTypes() []VertexType {
	buffer := make([]VertexType, 0, len(v))
	for i := range v {
		buffer = append(buffer, v[i].vType)
	}
	return buffer
}
func (v Vertices) families() []int {
	buffer := make([]int, len(v))
	for _, vtx := range v {
		for _, ance := range vtx.parents {
			buffer = append(buffer, ance.family)
		}
	}
	return buffer
}

func (v Vertices) parents() [][]KeyType {
	buffer := make([][]KeyType, len(v))
	for i, vtx := range v {
		for _, pred := range vtx.parents {
			buffer[i] = append(buffer[i], pred.msg)
		}
	}
	return buffer
}

func (v Vertices) unflatten() [][]*Vertex {
	sort.Sort(v)
	buffer := make([][]*Vertex, v[len(v)-1].batch)
	for i, vtx := range v {
		for _, pred := range vtx.parents {
			buffer[i] = append(buffer[i], pred)
		}
	}
	return buffer
}

func (vertices Vertices) Print() {
	for _, v := range vertices {
		v.Print()
		fmt.Println()
	}
}

type Vertexset [][]*Vertex

func (vtxSet Vertexset) flatten() []*Vertex {
	buffer := make([]*Vertex, 0, len(vtxSet))
	for i := 0; i < len(vtxSet); i++ {
		buffer = append(buffer, (vtxSet)[i]...)
	}
	return buffer
}

type VerticeByfamilys map[string]*Vertex

func (familys VerticeByfamilys) Unique() *map[int]*Vertex {
	buffer := map[int]*Vertex{}
	for _, vtx := range familys {
		if v, ok := buffer[vtx.family]; !ok || v.id < vtx.id {
			buffer[v.family] = vtx
		}
	}
	return &buffer
}

type VerticesByID []*Vertex

func (v VerticesByID) Len() int      { return len(v) }
func (v VerticesByID) Swap(i, j int) { v[i], v[j] = v[j], v[i] }
func (v VerticesByID) Less(i, j int) bool {
	return v[i].id < v[j].id
}

func (vertices Vertices) RemoveCopy(toRemove []*Vertex) []*Vertex {
	removalSet := make(map[*Vertex]bool)
	for _, v := range toRemove {
		removalSet[v] = true
	}

	newVertices := make([]*Vertex, 0, len(vertices))
	for _, v := range vertices {
		if _, ok := removalSet[v]; !ok {
			newVertices = append(newVertices, v)
		}
	}
	return newVertices
}

func QuickSort(vertices []*Vertex) []*Vertex {
	if len(vertices) < 2 {
		return vertices
	}
	left, right := 0, len(vertices)-1
	pivotIndex := rand.Int() % len(vertices)

	vertices[pivotIndex], vertices[right] = vertices[right], vertices[pivotIndex]
	for i := range vertices {
		if Compare(vertices[i], vertices[right]) {
			vertices[i], vertices[left] = vertices[left], vertices[i]
			left++
		}
	}
	vertices[left], vertices[right] = vertices[right], vertices[left]

	QuickSort(vertices[:left])
	QuickSort(vertices[left+1:])
	return vertices
}

func Compare(lft *Vertex, rgt *Vertex) bool {
	if lft.batch == rgt.batch {
		return lft.id < rgt.id
	}
	return lft.batch < rgt.batch
}
