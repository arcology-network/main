package scheduler

import (
	"bytes"

	"github.com/arcology-network/common-lib/common"
	eucommon "github.com/arcology-network/eu/common"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
	"github.com/arcology-network/main/modules/tools"
	mtypes "github.com/arcology-network/main/types"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type batch struct {
	context    *processContext
	sequences  []*mtypes.ExecutingSequence
	msgsToExec map[evmCommon.Hash]*schtyp.Message
}

// newBatch create the first batch for a generation.
func newBatch(context *processContext, sequences []*mtypes.ExecutingSequence) *batch {
	msgsToExec := make(map[evmCommon.Hash]*schtyp.Message)
	for _, seq := range sequences {
		for i := range seq.Txids {
			seq.Txids[i] = context.txId //<< 8
			context.txId++

			context.txHash2IdBiMap.Add(seq.Msgs[i].TxHash, seq.Txids[i])
		}

		if !seq.Parallel {
			msgsToExec[seq.SequenceId] = &schtyp.Message{
				Message: &eucommon.StandardMessage{
					TxHash: seq.SequenceId,
				},
			}
		} else {
			for _, msg := range seq.Msgs {
				msgsToExec[msg.TxHash] = &schtyp.Message{
					Message: msg,
				}
			}
		}
	}

	return &batch{
		context:    context,
		sequences:  sequences,
		msgsToExec: msgsToExec,
	}
}

// process handle the execution of the current batch, returns the next batch if defer calls exist.
func (b *batch) process(execTree *execTree) *batch {
	executed := b.setMsgPrecedings()
	// Process txs on executors.
	// responses, txHash2Spawned, seqId2Hashes, newContracts, deferHash2Id, deferHash2Callee :=
	responses, newContracts :=
		b.context.executor.Run(
			b.msgsToExec,
			b.sequences,
			b.context.timestamp,
			b.context.msgTemplate,
			b.context.logger,
			b.context.parallelism,
			b.context.generation,
			b.context.batch,
		)

	b.context.executed = append(b.context.executed, executed...)
	b.context.executedHash = tools.CalculateHash(b.context.executed)
	b.context.newContracts = append(b.context.newContracts, newContracts...)
	// Find conflictions.
	arbitrateParam := b.makeArbitrateParam(responses, execTree)
	conflictedHashes, cpLeft, cpRight := b.context.arbitrator.Do(
		arbitrateParam,
		b.context.logger,
		b.context.generation,
		b.context.batch,
	)

	cpLeft, cpRight = b.backtraceConflictionPairs(execTree, cpLeft, cpRight)
	for i := range cpLeft {
		ltxhash := b.context.txHash2IdBiMap.GetInverse(cpLeft[i])
		leftAddr, ok := b.context.txHash2Callee[ltxhash]
		if !ok {
			continue
		}
		leftSign, ok := b.context.txHash2Sign[ltxhash]
		if !ok {
			continue
		}

		rtxhash := b.context.txHash2IdBiMap.GetInverse(cpRight[i])
		rightAddr, ok := b.context.txHash2Callee[rtxhash]
		if !ok {
			continue
		}
		rightSign, ok := b.context.txHash2Sign[rtxhash]
		if !ok {
			continue
		}
		b.context.conflicts.Add(&mtypes.ConflictInfo{
			LeftAddress:  leftAddr,
			LeftSign:     leftSign,
			RightAddress: rightAddr,
			RightSign:    rightSign,
		})
	}

	deletedDict := make(map[evmCommon.Hash]struct{})
	for _, ch := range conflictedHashes {
		deletedDict[*ch] = struct{}{}
	}
	execTree.deleteBranches(deletedDict)
	common.MergeMaps(b.context.deletedDict, deletedDict)

	return nil
}

// func (b *batch) createsequenceId2Hashes() map[evmCommon.Hash][]evmCommon.Hash {
// 	seq2lst := make(map[evmCommon.Hash][]evmCommon.Hash, len(b.sequences))

// 	for _, seq := range b.sequences {
// 		lst := make([]evmCommon.Hash, 0, len(seq.Msgs))
// 		for _, msg := range seq.Msgs {
// 			if msg.Native.To != nil {
// 				b.context.txHash2Callee[msg.TxHash] = *msg.Native.To

// 			}
// 			lst = append(lst, msg.TxHash)
// 		}
// 		seq2lst[seq.SequenceId] = lst
// 	}
// 	return seq2lst
// }

// 1. Set `Precedings` and `PrecedingHash` for each message;
// 2. Update `context.txHash2Callee` for each message;
// 3. Collect parallel messages' hash, put them into `executed` and return.
func (b *batch) setMsgPrecedings() []*evmCommon.Hash {
	executed := make([]*evmCommon.Hash, 0, 50000)

	for _, seq := range b.sequences {
		for _, msg := range seq.Msgs {
			if msg.Native.To != nil {
				b.context.txHash2Callee[msg.TxHash] = *msg.Native.To
				sign := msg.Native.Data
				if len(msg.Native.Data) > 4 {
					sign = sign[:4]
				}
				b.context.txHash2Sign[msg.TxHash] = [4]byte(sign)
			}
			h := evmCommon.BytesToHash(msg.TxHash[:])
			executed = append(executed, &h)
		}

		if seq.Parallel {
			for _, msg := range seq.Msgs {
				schdMsg := b.msgsToExec[msg.TxHash]
				schdMsg.Precedings = &b.context.executed
				schdMsg.PrecedingHash = b.context.executedHash
				// executed = append(executed, &msg.TxHash)
			}
		} else {
			schdMsg := b.msgsToExec[seq.SequenceId]
			if schdMsg.DirectPrecedings != nil {
				precedings := make([]*evmCommon.Hash, 0, len(b.context.executedLastGen)+len(*schdMsg.DirectPrecedings))
				precedings = append(precedings, b.context.executedLastGen...)
				precedings = append(precedings, *schdMsg.DirectPrecedings...)
				schdMsg.Precedings = &precedings
				schdMsg.PrecedingHash = tools.CalculateHash(precedings)
			} else {
				schdMsg.Precedings = &b.context.executed
				schdMsg.PrecedingHash = b.context.executedHash
			}
		}
	}
	return executed
}

func (b *batch) makeArbitrateParam(
	responses map[evmCommon.Hash]*mtypes.ExecuteResponse,
	execTree *execTree,
) [][][]*mtypes.TxElement {

	var arbitrateParam [][][]*mtypes.TxElement
	// if len(deferId2Responses) == 0 {
	// The last batch, do arbitration among all the branches.
	branches := execTree.getBranches()
	if len(branches) > 1 {
		groups := make([][]*mtypes.TxElement, len(branches))
		for i, branch := range branches {
			group := make([]*mtypes.TxElement, 0, len(branch.layers[0])+len(branch.layers)-1)
			for lid, layer := range branch.layers {
				for j := range layer {
					group = append(group, &mtypes.TxElement{
						TxHash:  &layer[j],
						Batchid: uint64(lid),
						Txid:    b.context.txHash2IdBiMap.Get(layer[j]),
					})
				}
			}
			groups[i] = group
		}
		arbitrateParam = [][][]*mtypes.TxElement{groups}
	}

	(&schtyp.GasCache{DictionaryHash: b.context.txHash2Gas}).CostCalculateSort(&arbitrateParam)
	return arbitrateParam
}

func (b *batch) backtraceConflictionPairs(execTree *execTree, originL, originR []uint32) ([]uint32, []uint32) {
	resL := make([]uint32, 0, len(originL))
	resR := make([]uint32, 0, len(originR))
	// Tricky!
	getPrecedingsOrItself := func(layers [][]evmCommon.Hash, id uint32, hash evmCommon.Hash) []uint32 {
		// No defer.
		if len(layers) == 1 {
			return []uint32{id}
		}

		// Has defer.
		for _, nativeTx := range layers[0] {
			if bytes.Equal(nativeTx.Bytes(), hash.Bytes()) {
				return []uint32{id}
			}
		}

		precedingIds := make([]uint32, len(layers[0]))
		for i, preceding := range layers[0] {
			precedingIds[i] = b.context.txHash2IdBiMap.Get(preceding)
		}
		return precedingIds
	}
	for i := range originL {
		hl := b.context.txHash2IdBiMap.GetInverse(originL[i])
		hr := b.context.txHash2IdBiMap.GetInverse(originR[i])
		bl := execTree.getBranch(hl)
		br := execTree.getBranch(hr)
		lefts := getPrecedingsOrItself(bl.layers, originL[i], hl)
		rights := getPrecedingsOrItself(br.layers, originR[i], hr)
		for _, l := range lefts {
			for _, r := range rights {
				resL = append(resL, l)
				resR = append(resR, r)
			}
		}
	}
	return resL, resR
}
