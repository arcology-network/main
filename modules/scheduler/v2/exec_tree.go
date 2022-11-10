package scheduler

import (
	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmncmn "github.com/arcology-network/common-lib/common"
	cmntyp "github.com/arcology-network/common-lib/types"
)

type branch struct {
	id     ethcmn.Hash
	layers [][]ethcmn.Hash
}

func newBranch(id ethcmn.Hash, msgs []*cmntyp.StandardMessage) *branch {
	layer := make([]ethcmn.Hash, len(msgs))
	for i, m := range msgs {
		layer[i] = m.TxHash
	}

	return &branch{
		id:     id,
		layers: [][]ethcmn.Hash{layer},
	}
}

type execTree struct {
	id2Branch       map[ethcmn.Hash]*branch
	txHash2BranchId map[ethcmn.Hash]ethcmn.Hash
}

func newExecTree() *execTree {
	return &execTree{
		id2Branch:       make(map[ethcmn.Hash]*branch),
		txHash2BranchId: make(map[ethcmn.Hash]ethcmn.Hash),
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

func (tree *execTree) updateSequentialBranches(branches map[ethcmn.Hash][]ethcmn.Hash) {
	for seqId, msgs := range branches {
		if b, ok := tree.id2Branch[seqId]; ok {
			// Update the last layer.
			b.layers[len(b.layers)-1] = msgs
		} else {
			// FIXME: Is is possible?
			tree.id2Branch[seqId] = &branch{
				id:     seqId,
				layers: [][]ethcmn.Hash{msgs},
			}
		}

		for _, msg := range msgs {
			tree.txHash2BranchId[msg] = seqId
		}
	}
}

func (tree *execTree) deleteBranches(deletedDict map[ethcmn.Hash]struct{}) {
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

func (tree *execTree) mergeBranches(froms []*ethcmn.Hash, to ethcmn.Hash, id ethcmn.Hash) {
	for _, from := range froms {
		delete(tree.id2Branch, tree.txHash2BranchId[*from])
		tree.txHash2BranchId[*from] = id
	}

	tree.id2Branch[id] = &branch{
		id:     id,
		layers: [][]ethcmn.Hash{cmncmn.ToDereferencedSlice(froms), {to}},
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

func (tree *execTree) getBranch(hash ethcmn.Hash) *branch {
	if id, ok := tree.txHash2BranchId[hash]; ok {
		return tree.id2Branch[id]
	}
	return nil
}
