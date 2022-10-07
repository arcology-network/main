package deepgraph

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/HPISTechnologies/common-lib/common"
)

type Deepgraph struct {
	id            int
	batch         int
	vertices      map[KeyType]*Vertex
	mtx           *sync.RWMutex
	Total         int
	cachedLeavies []*Vertex
}

func (graph *Deepgraph) Get(key KeyType) *Vertex {
	return graph.vertices[key]
}

func (graph *Deepgraph) Set(key KeyType, value *Vertex) {
	graph.vertices[key] = value
}

func (graph *Deepgraph) Keys() []KeyType {
	keys := make([]KeyType, 0, len(graph.vertices))
	for k := range graph.vertices {
		keys = append(keys, k)

	}
	return keys
}

func (graph *Deepgraph) Values() []*Vertex {
	values := make([]*Vertex, 0, len(graph.vertices))
	for _, v := range graph.vertices {
		values = append(values, v)
	}
	return values
}

func (graph *Deepgraph) Delete(key KeyType) {
	delete(graph.vertices, key)
}

func (graph *Deepgraph) Size() int {
	return len(graph.vertices)
}
func (graph *Deepgraph) Reset() {
	id := graph.id
	*graph = *NewDeepgraph()
	graph.id = id
}

func (graph *Deepgraph) Ids(hashes []KeyType) []int {
	ids := make([]int, len(hashes))
	for i := range ids {
		ids[i] = graph.Size() + i
	}
	return ids
}

func NewDeepgraph() *Deepgraph {
	return &Deepgraph{
		id:       0,
		batch:    0,
		vertices: make(map[KeyType]*Vertex, 1024),
		mtx:      &sync.RWMutex{},
	}
}

/*-------------------------------------External interfaces --------------------------------------------------*/
func (graph *Deepgraph) Insert(msgHash []KeyType, ids []int, vType []VertexType, preds [][]KeyType) {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	vertices := VerticesFromSource(msgHash, ids, graph.batch, vType)
	for i, newVtx := range *vertices {
		newVtx.Link(graph.Find(preds[i]))
		graph.updateFamily(newVtx)
		graph.Set(msgHash[i], newVtx)
		newVtx.CacheAncestors()
	}

	graph.cachedLeavies = graph.retriveLeavies()
}

func (graph *Deepgraph) InsertEx(msgHash []KeyType, ids []int, vType []VertexType, preds [][]KeyType) {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	vertices := VerticesFromSource(msgHash, ids, graph.batch, vType)
	for i, newVtx := range *vertices {
		fmt.Println()
		t0 := time.Now()
		newVtx.Link(graph.Find(preds[i]))
		fmt.Println("deepgraph.Link():", time.Now().Sub(t0))
		t0 = time.Now()
		graph.updateFamily(newVtx)
		fmt.Println("deepgraph.updateFamily():", time.Now().Sub(t0))
		t0 = time.Now()
		graph.Set(msgHash[i], newVtx)
		fmt.Println("deepgraph.store():", time.Now().Sub(t0))
		t0 = time.Now()
		newVtx.CacheAncestors()
		fmt.Println("deepgraph.store():", time.Now().Sub(t0))
	}
	t0 := time.Now()
	graph.cachedLeavies = graph.retriveLeavies()
	fmt.Println("deepgraph.retriveLeavies():", time.Now().Sub(t0))
	fmt.Println()
}

func (graph *Deepgraph) GetAncestors(hashes []KeyType) ([][]KeyType, [][]int, [][]int) {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	export := func(vtx []*Vertex) ([]KeyType, []int, []int) {
		return append(vtx[0].cachedAnceKeys, vtx[0].msg), append(vtx[0].cachedAnceBatches, vtx[0].batch), append(vtx[0].cachedAnceFamilies, vtx[0].family)
	}

	return graph.reformat(hashes, export, func(vtx *Vertex) []*Vertex { return []*Vertex{vtx} })
}

// not in use for now
func (graph *Deepgraph) GetDescendents(hashes []KeyType) ([][]KeyType, [][]int, [][]int) {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	return graph.reformat(hashes, Vertices{}.Export, graph.retriveDescendents)
}

//For arbitration
func (graph *Deepgraph) GetAncestorFamilies(hashes []KeyType) ([][]KeyType, [][]int, [][]int) {
	return graph.GetAncestors(hashes)
}

// For reverting transactions trees after conflicting states have been detected
// func (graph *Deepgraph) GetDescendentFamilies(hashes []KeyType) ([][]KeyType, [][]int, [][]int) {
// 	graph.mtx.Lock()
// 	defer graph.mtx.Unlock()

// 	return graph.reformat(hashes, Vertices{}.Export, graph.retriveDescendentFamilies)
// }

// Move all vertices from one graph to another
// func (graph *Deepgraph) MoveFrom(from *Deepgraph) {
// 	graph.mtx.Lock()
// 	defer graph.mtx.Unlock()

// 	for _, batch := range from.batches() {
// 		graph.Insert(
// 			Vertices(batch).keys(),
// 			graph.Ids(Vertices(batch).keys()),
// 			Vertices(batch).vTypes(),
// 			Vertices(batch).parents())
// 	}
// 	from.Reset()
// }

// Get all the families
func (graph *Deepgraph) GetSubgraphs() ([][]KeyType, [][]int, [][]int) {
	return graph.GetAncestors(Vertices(graph.cachedLeavies).keys())
}

func (graph *Deepgraph) GetAll() []KeyType {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	buffer := make([]*Vertex, 0, graph.Size())
	for _, v := range graph.Values() {
		buffer = append(buffer, v)
	}

	sort.Sort(Vertices(buffer))
	return Vertices(buffer).keys()
}

func (graph *Deepgraph) Remove(hashes []KeyType) {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	graph.cachedLeavies = Vertices(graph.cachedLeavies).RemoveCopy(graph.Find(hashes))
	graph.remove(graph.Find(hashes))
}

func (graph *Deepgraph) GetLeavies() []KeyType {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	return Vertices(graph.retriveLeavies()).keys()
}

func (graph *Deepgraph) IncreaseBatch() {
	graph.mtx.Lock()
	defer graph.mtx.Unlock()

	graph.batch++
}

/*--------------------------------------------End-------------------------------------------------*/
func (graph *Deepgraph) reformat(
	hashes []KeyType,
	export func([]*Vertex) ([]KeyType, []int, []int),
	action func(vtx *Vertex) []*Vertex) ([][]KeyType, [][]int, [][]int) {
	buffer, batch, family := make([][]KeyType, len(hashes)), make([][]int, len(hashes)), make([][]int, len(hashes))

	worker := func(start, end, idx int, args ...interface{}) {
		graph := args[0].([]interface{})[0].(*Deepgraph)
		hashes := args[0].([]interface{})[1].([]KeyType)
		buffer := args[0].([]interface{})[2].([][]KeyType)
		batch := args[0].([]interface{})[3].([][]int)
		family := args[0].([]interface{})[4].([][]int)

		action := args[0].([]interface{})[5].(func(vtx *Vertex) []*Vertex)

		for i, v := range graph.Find(hashes[start:end]) {
			buffer[i+start], batch[i+start], family[i+start] = export(action(v))
		}
	}

	common.ParallelWorker(len(hashes), 8, worker, graph, hashes, buffer, batch, family, action)
	return buffer, batch, family
}

// return all vertices in the same batch as they were inserted
func (graph *Deepgraph) batches() [][]*Vertex {
	buffer := make([][]*Vertex, graph.Size())

	for _, vtx := range graph.Values() {
		buffer[vtx.batch] = append(buffer[vtx.batch], vtx)
	}

	for i := range buffer {
		sort.Sort(Vertices(buffer[i]))
	}
	return buffer
}

func (graph *Deepgraph) Find(hashes []KeyType) []*Vertex {
	if len(hashes) > 65536 {
		return graph.ParallelFind(hashes)
	}
	return graph.SerialFind(hashes)
}

func (graph *Deepgraph) ParallelFind(hashes []KeyType) []*Vertex {
	t0 := time.Now()
	buffer := make([]*Vertex, len(hashes))
	worker := func(start int, end int, idx int, arg ...interface{}) {
		for j := start; j < end; j++ {
			buffer[j] = graph.Get(hashes[j])
		}
	}
	common.ParallelWorker(len(hashes), 16, worker)
	fmt.Println("deepgraph.Find():", time.Now().Sub(t0))
	return buffer
}

func (graph *Deepgraph) SerialFind(hashes []KeyType) []*Vertex {
	buffer := make([]*Vertex, len(hashes))
	for j := range hashes {
		buffer[j] = graph.Get(hashes[j])
	}
	return buffer
}

func (graph *Deepgraph) retrive(vtx *Vertex, direction Direction) []*Vertex {
	action := func(vtx *Vertex, direction Direction, buffer []*Vertex) []*Vertex {
		vertices := direction(vtx, buffer)
		for _, v := range vertices {
			if v != nil {
				buffer = append(buffer, v)
			}
		}
		return vertices
	}

	buffer := []*Vertex{}
	vtx.traverse(buffer, direction, action)

	array := make([]*Vertex, 0, len(buffer)+1)
	for _, v := range buffer {
		array = append(array, v)
	}

	array = append(array, vtx)
	sort.Sort(Vertices(array))
	return array
}

func (graph *Deepgraph) retriveAncestors(vtx *Vertex) []*Vertex {
	up := func(from *Vertex, buffer []*Vertex) []*Vertex {
		return from.parents
	}
	return graph.retrive(vtx, up)
}

func (graph *Deepgraph) retriveDescendents(vtx *Vertex) []*Vertex {
	down := func(from *Vertex, buffer []*Vertex) []*Vertex {
		return []*Vertex{from.child}
	}
	return graph.retrive(vtx, down)
}

// func (graph *Deepgraph) retriveAncestorFamilies(vtx *Vertex) []*Vertex {
// 	up := func(from *Vertex) []*Vertex {
// 		return from.parents
// 	}
// 	return graph.retriveFamilyPaths(vtx, up)
// }

// func (graph *Deepgraph) retriveDescendentFamilies(vtx *Vertex) []*Vertex {
// 	down := func(from *Vertex) []*Vertex {
// 		return []*Vertex{from.child}
// 	}
// 	return graph.retriveFamilyPaths(vtx, down)
// }

// func (graph *Deepgraph) retriveFamilyPaths(vtx *Vertex, direction func(vtx *Vertex) []*Vertex) []*Vertex {
// 	buffer := graph.familyMembers(vtx)
// 	if len(buffer) == 1 { // the only family member
// 		for _, rep := range direction(buffer[0]) {
// 			if vertices := graph.retriveFamilyPaths(rep, direction); vertices != nil {
// 				buffer = append(buffer, vertices...)
// 			}
// 		}
// 	} else {
// 		// get all vertices of the family
// 		vertices := make(map[KeyType]*Vertex)
// 		for _, member := range buffer {
// 			for k, pred := range direction(member) {
// 				vertices[k] = pred
// 			}
// 		}

// 		// get representatives for each family
// 		reps := map[int]*Vertex{}
// 		for _, vtx := range vertices {
// 			if v, ok := reps[vtx.family]; !ok || v.id < vtx.id {
// 				reps[vtx.family] = vtx
// 			}
// 		}

// 		families := make([][]*Vertex, 0, len(reps))
// 		for k, rep := range reps {
// 			if k != vtx.family {
// 				if vertices := graph.retriveFamilyPaths(rep, direction); vertices != nil {
// 					families = append(families, vertices)
// 				}
// 			}
// 		}
// 		buffer = append(buffer, Vertexset(families).flatten()...)
// 	}
// 	sort.Sort(Vertices(buffer))
// 	return buffer
// }

func (graph *Deepgraph) retriveLeavies() []*Vertex {
	buffer := make([]*Vertex, 0, graph.Size())

	for _, vtx := range graph.Values() {
		if vtx.child == nil {
			buffer = append(buffer, vtx)
		}
	}
	QuickSort(Vertices(buffer))
	return buffer
}

// all vertices under the same family id
func (graph *Deepgraph) familyMembers(vtx *Vertex) []*Vertex {
	if vtx.vType == Original && vtx.id == vtx.family {
		return []*Vertex{vtx}
	}

	buffer := make([]*Vertex, 0, graph.Size())
	for _, v := range graph.Values() {
		if vtx.family == v.family {
			buffer = append(buffer, v)
		}
	}
	sort.Sort(Vertices(buffer))
	return buffer
}

// bind under current vertex, spawned only
func (graph *Deepgraph) updateFamily(current *Vertex) {
	if current == nil || current.vType != Spawned {
		return
	}

	targetIDs := make(map[int]int)
	for _, ancestor := range current.parents {
		if !(ancestor.vType == Original && ancestor.id == ancestor.family) {
			// targetIDs = append(targetIDs, ancestor.family)
			targetIDs[ancestor.family] = ancestor.family
		} else {
			ancestor.family = current.family
		}
	}

	for _, id := range targetIDs {
		for _, vtx := range graph.Values() {
			if vtx.family == id {
				vtx.family = current.family // bind the vertex under the current one
			}
		}
	}
}

func (graph *Deepgraph) remove(vertices []*Vertex) {
	for _, v := range vertices {
		if v != nil {
			graph.Delete(v.msg)
		}
	}
}

func (graph *Deepgraph) printVertices() {
	for _, vtx := range graph.Values() {
		vtx.Print()
		fmt.Println()
	}
	fmt.Println(" ========== =============")
}
