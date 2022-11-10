package scheduler

import (
	"bytes"
	"crypto/sha256"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	ethtyp "github.com/arcology-network/3rd-party/eth/types"
	cmncmn "github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/mhasher"
	cmntyp "github.com/arcology-network/common-lib/types"
	schtyp "github.com/arcology-network/main/modules/scheduler/types"
)

type batch struct {
	context    *processContext
	sequences  []*cmntyp.ExecutingSequence
	msgsToExec map[ethcmn.Hash]*schtyp.Message
}

// newBatch create the first batch for a generation.
func newBatch(context *processContext, sequences []*cmntyp.ExecutingSequence) *batch {
	msgsToExec := make(map[ethcmn.Hash]*schtyp.Message)
	for _, seq := range sequences {
		for i := range seq.Txids {
			seq.Txids[i] = context.txId << 8
			context.txId++

			context.txHash2IdBiMap.Add(seq.Msgs[i].TxHash, seq.Txids[i])
		}

		if !seq.Parallel {
			msgsToExec[seq.SequenceId] = &schtyp.Message{
				Message: &cmntyp.StandardMessage{
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
	responses, txHash2Spawned, seqId2Hashes, newContracts, deferHash2Id, deferHash2Callee :=
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

	// Update context's status.
	for hash, spawned := range txHash2Spawned {
		b.context.spawnedRelations = append(b.context.spawnedRelations, &cmntyp.SpawnedRelation{
			Txhash:        hash,
			SpawnedTxHash: spawned,
		})
	}
	for _, hashes := range seqId2Hashes {
		executed = append(executed, cmncmn.ToReferencedSlice(hashes)...)
	}
	b.context.executed = append(b.context.executed, executed...)
	b.context.executedHash = cmncmn.CalculateHash(b.context.executed)
	execTree.updateSequentialBranches(seqId2Hashes)
	b.context.newContracts = append(b.context.newContracts, newContracts...)
	for hash, id := range deferHash2Id {
		b.context.txHash2IdBiMap.Add(hash, id)
	}
	cmncmn.MergeMaps(b.context.txHash2Callee, deferHash2Callee)

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
		if al, ok := b.context.txHash2Callee[b.context.txHash2IdBiMap.GetInverse(cpLeft[i])]; ok {
			if ar, ok := b.context.txHash2Callee[b.context.txHash2IdBiMap.GetInverse(cpRight[i])]; ok {
				b.context.conflicts[al] = append(b.context.conflicts[al], ar)
			}
		}
	}

	deletedDict := make(map[ethcmn.Hash]struct{})
	for _, ch := range conflictedHashes {
		deletedDict[*ch] = struct{}{}
	}
	execTree.deleteBranches(deletedDict)
	cmncmn.MergeMaps(b.context.deletedDict, deletedDict)

	// Construct defer calls.
	deferId2Responses := make(map[string][]*cmntyp.ExecuteResponse)
	for _, resp := range responses {
		if resp.DfCall == nil || resp.Status == ethtyp.ReceiptStatusFailed {
			continue
		}

		if _, ok := deletedDict[resp.Hash]; ok {
			continue
		}

		deferId := string(resp.DfCall.ContractAddress) + resp.DfCall.DeferID
		deferId2Responses[deferId] = append(deferId2Responses[deferId], resp)
	}
	if len(deferId2Responses) != 0 {
		return b.createDeferBatch(deferId2Responses, execTree)
	}
	return nil
}

// 1. Set `Precedings` and `PrecedingHash` for each message;
// 2. Update `context.txHash2Callee` for each message;
// 3. Collect parallel messages' hash, put them into `executed` and return.
func (b *batch) setMsgPrecedings() []*ethcmn.Hash {
	executed := make([]*ethcmn.Hash, 0, 50000)
	for _, seq := range b.sequences {
		for _, msg := range seq.Msgs {
			if msg.Native.To() != nil {
				b.context.txHash2Callee[msg.TxHash] = *msg.Native.To()
			}
		}

		if seq.Parallel {
			for _, msg := range seq.Msgs {
				schdMsg := b.msgsToExec[msg.TxHash]
				schdMsg.Precedings = &b.context.executed
				schdMsg.PrecedingHash = b.context.executedHash
				executed = append(executed, &msg.TxHash)
			}
		} else {
			schdMsg := b.msgsToExec[seq.SequenceId]
			if schdMsg.DirectPrecedings != nil {
				precedings := make([]*ethcmn.Hash, 0, len(b.context.executedLastGen)+len(*schdMsg.DirectPrecedings))
				precedings = append(precedings, b.context.executedLastGen...)
				precedings = append(precedings, *schdMsg.DirectPrecedings...)
				schdMsg.Precedings = &precedings
				schdMsg.PrecedingHash = cmncmn.CalculateHash(precedings)
			} else {
				schdMsg.Precedings = &b.context.executed
				schdMsg.PrecedingHash = b.context.executedHash
			}
		}
	}
	return executed
}

func (b *batch) makeArbitrateParam(
	responses map[ethcmn.Hash]*cmntyp.ExecuteResponse,
	execTree *execTree,
) [][][]*cmntyp.TxElement {
	// Aggregate defer calls.
	deferId2Responses := make(map[string][]*cmntyp.ExecuteResponse)
	for _, resp := range responses {
		b.context.txHash2Gas[resp.Hash] = resp.GasUsed

		if resp.DfCall == nil || resp.Status == ethtyp.ReceiptStatusFailed {
			continue
		}
		// A unique defer id is composed of contract address and defer id.
		deferId := string(resp.DfCall.ContractAddress) + resp.DfCall.DeferID
		deferId2Responses[deferId] = append(deferId2Responses[deferId], resp)
	}

	var arbitrateParam [][][]*cmntyp.TxElement
	if len(deferId2Responses) == 0 {
		// The last batch, do arbitration among all the branches.
		branches := execTree.getBranches()
		if len(branches) > 1 {
			groups := make([][]*cmntyp.TxElement, len(branches))
			for i, branch := range branches {
				group := make([]*cmntyp.TxElement, 0, len(branch.layers[0])+len(branch.layers)-1)
				for lid, layer := range branch.layers {
					for j := range layer {
						group = append(group, &cmntyp.TxElement{
							TxHash:  &layer[j],
							Batchid: uint64(lid),
							Txid:    b.context.txHash2IdBiMap.Get(layer[j]),
						})
					}
				}
				groups[i] = group
			}
			arbitrateParam = [][][]*cmntyp.TxElement{groups}
		}
	} else {
		// Do arbitration in each defer group.
		for _, responses := range deferId2Responses {
			if len(responses) == 1 {
				// If the group include 1 tx only, then no need to do arbitration.
				continue
			}

			groups := make([][]*cmntyp.TxElement, len(responses))
			for i := range responses {
				groups[i] = []*cmntyp.TxElement{
					{
						TxHash:  &responses[i].Hash,
						Batchid: 0,
						Txid:    b.context.txHash2IdBiMap.Get(responses[i].Hash),
					},
				}
			}
			arbitrateParam = append(arbitrateParam, groups)
		}
	}

	(&schtyp.GasCache{DictionaryHash: b.context.txHash2Gas}).CostCalculateSort(&arbitrateParam)
	return arbitrateParam
}

func (b *batch) backtraceConflictionPairs(execTree *execTree, originL, originR []uint32) ([]uint32, []uint32) {
	resL := make([]uint32, 0, len(originL))
	resR := make([]uint32, 0, len(originR))
	// Tricky!
	getPrecedingsOrItself := func(layers [][]ethcmn.Hash, id uint32, hash ethcmn.Hash) []uint32 {
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

func (b *batch) createDeferBatch(
	deferId2Responses map[string][]*cmntyp.ExecuteResponse,
	execTree *execTree,
) *batch {
	sequences := make([]*cmntyp.ExecutingSequence, 0, len(deferId2Responses))
	msgsToExec := make(map[ethcmn.Hash]*schtyp.Message)
	for _, responses := range deferId2Responses {
		hashBytes := make([][]byte, len(responses))
		precedings := make([]*ethcmn.Hash, len(responses))
		maxTxId := uint32(0)
		for i, resp := range responses {
			hashBytes[i] = resp.Hash.Bytes()
			precedings[i] = &resp.Hash

			txId := b.context.txHash2IdBiMap.Get(resp.Hash)
			if txId > maxTxId {
				maxTxId = txId
			}
		}

		sortedBytes, _ := mhasher.SortBytes(hashBytes)
		spawnedHash := sha256.Sum256(cmncmn.Flatten(sortedBytes))
		for _, preceding := range precedings {
			b.context.spawnedRelations = append(b.context.spawnedRelations, &cmntyp.SpawnedRelation{
				Txhash:        *preceding,
				SpawnedTxHash: spawnedHash,
			})
		}

		stdMsg := cmntyp.MakeMessageWithDefCall(responses[0].DfCall, spawnedHash, 0)
		seq := cmntyp.NewExecutingSequence([]*cmntyp.StandardMessage{stdMsg}, false)
		seq.Txids = []uint32{maxTxId + 1}
		sequences = append(sequences, seq)
		msgsToExec[seq.SequenceId] = &schtyp.Message{
			Message:          stdMsg,
			IsSpawned:        true,
			DirectPrecedings: &precedings,
		}

		b.context.txHash2IdBiMap.Add(spawnedHash, seq.Txids[0])
		execTree.mergeBranches(precedings, stdMsg.TxHash, seq.SequenceId)
	}

	return &batch{
		context:    b.context,
		sequences:  sequences,
		msgsToExec: msgsToExec,
	}
}
