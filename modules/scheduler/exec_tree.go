package scheduler

import (
	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	evmCommon "github.com/arcology-network/evm/common"
)

type branch struct {
	id     evmCommon.Hash
	layers [][]evmCommon.Hash
}

func newBranch(id evmCommon.Hash, msgs []*cmntyp.StandardMessage) *branch {
	layer := make([]evmCommon.Hash, len(msgs))
	for i, m := range msgs {
		layer[i] = m.TxHash
	}

	return &branch{
		id:     id,
		layers: [][]evmCommon.Hash{layer},
	}
}

type execTree struct {
	id2Branch       map[evmCommon.Hash]*branch
	txHash2BranchId map[evmCommon.Hash]evmCommon.Hash
}

func newExecTree() *execTree {
	return &execTree{
		id2Branch:       make(map[evmCommon.Hash]*branch),
		txHash2BranchId: make(map[evmCommon.Hash]evmCommon.Hash),
	}
}

func (tree *execTree) createBranches(sequences []*cmntyp.ExecutingSequence) {
	for _, seq := range sequences {
		if seq.Parallel {
			for _, msg := range seq.Msgs {
				tree.id2Branch[msg.TxHash] = newBranch(msg.TxHash, []*cmntyp.StandardMessage{msg})
				tree.txHash2BranchId[msg.TxHash] = msg.TxHash
			}
		} else {
			tree.id2Branch[seq.SequenceId] = newBranch(seq.SequenceId, seq.Msgs)
			for _, msg := range seq.Msgs {
				tree.txHash2BranchId[msg.TxHash] = seq.SequenceId
			}
		}
	}
}

func (tree *execTree) updateSequentialBranches(branches map[evmCommon.Hash][]evmCommon.Hash) {
	for seqId, msgs := range branches {
		if b, ok := tree.id2Branch[seqId]; ok {
			// Update the last layer.
			b.layers[len(b.layers)-1] = msgs
		} else {
			// FIXME: Is is possible?
			tree.id2Branch[seqId] = &branch{
				id:     seqId,
				layers: [][]evmCommon.Hash{msgs},
			}
		}

		for _, msg := range msgs {
			tree.txHash2BranchId[msg] = seqId
		}
	}
}

func (tree *execTree) deleteBranches(deletedDict map[evmCommon.Hash]struct{}) {
	for hash := range deletedDict {
		b := tree.getBranch(hash)
		if b == nil {
			continue
		}

		for _, layer := range b.layers {
			for _, member := range layer {
				delete(tree.txHash2BranchId, member)
			}
		}
		delete(tree.id2Branch, b.id)
	}
}

func (tree *execTree) mergeBranches(froms []*evmCommon.Hash, to evmCommon.Hash, id evmCommon.Hash) {
	for _, from := range froms {
		delete(tree.id2Branch, tree.txHash2BranchId[*from])
		tree.txHash2BranchId[*from] = id
	}

	tree.id2Branch[id] = &branch{
		id:     id,
		layers: [][]evmCommon.Hash{cmncmn.ToDereferencedSlice(froms), {to}},
	}
	tree.txHash2BranchId[to] = id
}

func (tree *execTree) getBranches() []*branch {
	res := make([]*branch, 0, len(tree.id2Branch))
	for _, branch := range tree.id2Branch {
		res = append(res, branch)
	}
	return res
}

func (tree *execTree) getBranch(hash evmCommon.Hash) *branch {
	if id, ok := tree.txHash2BranchId[hash]; ok {
		return tree.id2Branch[id]
	}
	return nil
}
