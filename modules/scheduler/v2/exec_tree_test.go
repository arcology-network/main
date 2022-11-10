package scheduler

import (
	"bytes"
	"testing"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmntyp "github.com/arcology-network/common-lib/types"
)

func TestExecTreeBasic(t *testing.T) {
	tree := newExecTree()
	sequences := []*cmntyp.ExecutingSequence{
		cmntyp.NewExecutingSequence([]*cmntyp.StandardMessage{
			{
				TxHash: ethcmn.BytesToHash([]byte{1}),
			},
		}, true),
		cmntyp.NewExecutingSequence([]*cmntyp.StandardMessage{
			{
				TxHash: ethcmn.BytesToHash([]byte{2}),
			},
		}, true),
		cmntyp.NewExecutingSequence([]*cmntyp.StandardMessage{
			{
				TxHash: ethcmn.BytesToHash([]byte{3}),
			},
			{
				TxHash: ethcmn.BytesToHash([]byte{4}),
			},
		}, false),
		cmntyp.NewExecutingSequence([]*cmntyp.StandardMessage{
			{
				TxHash: ethcmn.BytesToHash([]byte{5}),
			},
		}, true),
		cmntyp.NewExecutingSequence([]*cmntyp.StandardMessage{
			{
				TxHash: ethcmn.BytesToHash([]byte{6}),
			},
		}, true),
	}

	tree.createBranches(sequences)
	branches := tree.getBranches()
	if len(branches) != 5 {
		t.Errorf("num of branches, expected 5, got %d", len(branches))
		return
	}
	branch := tree.getBranch(ethcmn.BytesToHash([]byte{1}))
	if !bytes.Equal(branch.id.Bytes(), ethcmn.BytesToHash([]byte{1}).Bytes()) {
		t.Errorf("get parallel branch error.")
		return
	}
	branch = tree.getBranch(ethcmn.BytesToHash([]byte{3}))
	if !bytes.Equal(branch.id.Bytes(), sequences[2].SequenceId.Bytes()) {
		t.Errorf("get sequential branch error.")
		return
	}

	tree.updateSequentialBranches(map[ethcmn.Hash][]ethcmn.Hash{
		sequences[2].SequenceId: {
			ethcmn.BytesToHash([]byte{3}),
			ethcmn.BytesToHash([]byte{3, 1}),
			ethcmn.BytesToHash([]byte{4}),
			ethcmn.BytesToHash([]byte{4, 1}),
		},
	})
	t.Log("after updateSequentialBranches.")
	if len(tree.id2Branch) != 5 {
		t.Errorf("num of branches, expected 5, got %d", len(tree.id2Branch))
		return
	}
	if len(tree.txHash2BranchId) != 8 {
		t.Errorf("num of txs, expected 8, got %d", len(tree.txHash2BranchId))
		return
	}
	branch = tree.getBranch(ethcmn.BytesToHash([]byte{3, 1}))
	if !bytes.Equal(branch.id.Bytes(), sequences[2].SequenceId.Bytes()) {
		t.Errorf("get sequential branch error.")
		return
	}

	froms := []ethcmn.Hash{ethcmn.BytesToHash([]byte{5}), ethcmn.BytesToHash([]byte{6})}
	tree.mergeBranches(
		[]*ethcmn.Hash{&froms[0], &froms[1]},
		ethcmn.BytesToHash([]byte{5, 6}),
		sequences[4].SequenceId,
	)
	t.Log("after mergeBranches.")
	if len(tree.id2Branch) != 4 {
		t.Errorf("num of branches, expected 4, got %d", len(tree.id2Branch))
		return
	}
	if len(tree.txHash2BranchId) != 9 {
		t.Errorf("num of txs, expected 9, got %d", len(tree.txHash2BranchId))
	}
	branch = tree.getBranch(ethcmn.BytesToHash([]byte{5}))
	if !bytes.Equal(branch.id.Bytes(), sequences[4].SequenceId.Bytes()) ||
		len(branch.layers) != 2 ||
		len(branch.layers[0]) != 2 ||
		len(branch.layers[1]) != 1 {
		t.Errorf("get merged branch error.")
	}
}
