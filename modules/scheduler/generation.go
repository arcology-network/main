package scheduler

type generation struct {
	context  *processContext
	batches  []*batch
	execTree *execTree
}

func newGeneration(context *processContext, batches []*batch) *generation {
	execTree := newExecTree()
	execTree.createBranches(batches[0].sequences)

	return &generation{
		context:  context,
		batches:  batches,
		execTree: execTree,
	}
}

func (gen *generation) process() {
	nextBatch := gen.batches[0]
	for {
		gen.context.onNewBatch()
		nextBatch = nextBatch.process(gen.execTree)
		if nextBatch != nil {
			gen.batches = append(gen.batches, nextBatch)
		} else {
			return
		}
	}
}
