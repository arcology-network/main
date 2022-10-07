package scheduler

import (
	"crypto/sha256"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	ethTypes "github.com/HPISTechnologies/3rd-party/eth/types"
	"github.com/HPISTechnologies/common-lib/common"
	"github.com/HPISTechnologies/common-lib/mhasher"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/log"
	deepgraph "github.com/HPISTechnologies/main/modules/scheduler/deepgraph"
	schedulingTypes "github.com/HPISTechnologies/main/modules/scheduler/types"
	"go.uber.org/zap"
)

type DependencyManager struct {
	workGraph        *deepgraph.Deepgraph
	batchid          int64
	MainHashs        []*ethCommon.Hash
	removedTxs       *dataPool
	spawnedTxs       *dataPool
	executedTxs      *dataPool
	spawnedRelations []*types.SpawnedRelation
	arbitrator       Arbitrator
	innerlog         *actor.WorkerThreadLogger
	gasCache         *schedulingTypes.GasCache
	concurrency      int
	sequenceLookup   *schedulingTypes.SequenceLookup
	sequences        []*types.ExecutingSequence
	txidLookup       *schedulingTypes.TxIdsLookup
	txAddressLookup  *schedulingTypes.TxAddressLookup
	conflictlist     *schedulingTypes.ConflictList
}

//return a Subscriber struct
func NewDependencyManager(arbitrator Arbitrator, concurrency int) *DependencyManager {
	txidLookup := schedulingTypes.NewTxIdsLookup()
	txAddressLookup := schedulingTypes.NewTxAddressLookup()

	return &DependencyManager{
		MainHashs:        []*ethCommon.Hash{},
		removedTxs:       newDataPool(),
		spawnedTxs:       newDataPool(),
		executedTxs:      newDataPool(),
		arbitrator:       arbitrator,
		gasCache:         schedulingTypes.NewGasCache(),
		concurrency:      concurrency,
		spawnedRelations: []*types.SpawnedRelation{},
		sequenceLookup:   schedulingTypes.NewSequenceLookup(),
		sequences:        []*types.ExecutingSequence{},
		txidLookup:       txidLookup,
		txAddressLookup:  txAddressLookup,
		conflictlist:     schedulingTypes.NewConflictList(txAddressLookup, txidLookup),
	}
}

func (dm *DependencyManager) Start() {
	dm.workGraph = deepgraph.NewDeepgraph()
	dm.batchid = -1
	dm.arbitrator.Start()
}

func (dm *DependencyManager) Stop() {
	dm.arbitrator.Stop()
}

func getPrecedingHash(hashes []*ethCommon.Hash) ethCommon.Hash {
	if len(hashes) == 0 {
		return ethCommon.Hash{}
	}

	bys2D := [][]byte{}
	for _, h := range hashes {
		bys2D = append(bys2D, h.Bytes())
	}
	return mhasher.GetTxsHash(bys2D)
}

func convertKeyTypes(hashs []*ethCommon.Hash) []deepgraph.KeyType {
	keyTypes := make([]deepgraph.KeyType, len(hashs))
	for i := range hashs {
		keyTypes[i] = deepgraph.KeyType(*hashs[i])
	}
	return keyTypes
}

func convertHash(two []deepgraph.KeyType) []*ethCommon.Hash {
	twoHashs := make([]*ethCommon.Hash, len(two))
	for i, keytyps := range two {
		hash := ethCommon.Hash(keytyps)
		twoHashs[i] = &hash
	}
	return twoHashs
}

func mergeHashList(one, two []*ethCommon.Hash) []*ethCommon.Hash {
	if len(one) == 0 {
		return two
	}
	if len(two) == 0 {
		return one
	}
	returnHashes := append(one, two...)
	return returnHashes
}

func (dm *DependencyManager) insertIntoGraph(msgs map[ethCommon.Hash]*schedulingTypes.Message) error {
	hashs := make([]deepgraph.KeyType, len(msgs))
	txTypes := make([]deepgraph.VertexType, len(msgs))
	precedings := make([][]deepgraph.KeyType, len(msgs))
	dm.innerlog.Log(log.LogLevel_Debug, "prepare insertIntoGraph msgs into tree >>>>>>>>>>>>>>>>>>>>>>>>>>>>>", zap.Int("msgs", len(msgs)))
	i := 0
	for _, msg := range msgs {
		directPrecedings := []deepgraph.KeyType{}
		if msg.DirectPrecedings != nil {
			directPrecedings = make([]deepgraph.KeyType, len(*msg.DirectPrecedings))
			for j, hash := range *msg.DirectPrecedings {
				directPrecedings[j] = deepgraph.KeyType(*hash)
			}
		}
		precedings[i] = directPrecedings

		if msg.IsSpawned {
			txTypes[i] = deepgraph.Spawned
		} else {
			txTypes[i] = deepgraph.Original
		}
		hashs[i] = deepgraph.KeyType(msg.Message.TxHash)
		i = i + 1
	}

	dm.innerlog.Log(log.LogLevel_Debug, "start insertIntoGraph msgs into tree >>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	dm.workGraph.Insert(hashs, dm.workGraph.Ids(hashs), txTypes, precedings)
	dm.workGraph.IncreaseBatch()
	dm.innerlog.Log(log.LogLevel_Debug, "insertIntoGraph msgs into tree ", zap.Int("node length", len(hashs)))
	return nil
}

func (dm *DependencyManager) OnNewBatch(msgs map[ethCommon.Hash]*schedulingTypes.Message, broker *actor.MessageWrapper, inlog *actor.WorkerThreadLogger, sequences []*types.ExecutingSequence) bool {
	dm.txidLookup.Adds(sequences)
	dm.innerlog = inlog
	if len(msgs) == 0 {
		accumulatedExecutedTxs := dm.executedTxs.getData()
		accumulatedRemovedTxs := dm.removedTxs.getData()

		inclusiveList := getInclusiveList(accumulatedExecutedTxs, accumulatedRemovedTxs)

		broker.Send(actor.MsgInclusive, inclusiveList)
		broker.Send(actor.MsgSpawnedRelations, dm.spawnedRelations)
		return true
	} else {
		for _, msg := range msgs {
			if msg.IsSpawned == false {
				dm.MainHashs = mergeHashList(dm.MainHashs, dm.sequenceLookup.Zoomin(convertHash(dm.workGraph.GetAll())))
				dm.workGraph = deepgraph.NewDeepgraph()
				dm.gasCache = schedulingTypes.NewGasCache()
				dm.sequenceLookup.Clear()
				break
			}
		}
	}

	dm.sequences = sequences
	dm.batchid = dm.batchid + 1
	//setup relation between sequence and msg
	for _, sequence := range sequences {
		if sequence.Parallel {
			for _, msg := range sequence.Msgs {
				dm.sequenceLookup.Add(msg.TxHash, []ethCommon.Hash{msg.TxHash})
			}
		} else {
			hashList := make([]ethCommon.Hash, len(sequence.Msgs))
			for i, msg := range sequence.Msgs {
				hashList[i] = msg.TxHash
			}
			dm.sequenceLookup.Add(sequence.SequenceId, hashList)
		}
	}

	for _, sequence := range sequences {
		if sequence.Parallel {
			//include spawned
			if len(sequence.Msgs) == 0 {
				continue
			}
			if firstMsg, ok := msgs[sequence.Msgs[0].TxHash]; ok {
				if firstMsg.DirectPrecedings != nil && len(*firstMsg.DirectPrecedings) > 0 {
					for _, msg := range sequence.Msgs {
						currentPrecedings := []deepgraph.KeyType{}
						schedulingMsg, okk := msgs[msg.TxHash]
						if !okk {
							continue
						}
						if schedulingMsg.DirectPrecedings != nil && len(*schedulingMsg.DirectPrecedings) > 0 {
							//spawned txs
							//query all precedings from tree
							keyTypes, _, _ := dm.workGraph.GetAncestors(convertKeyTypes(*schedulingMsg.DirectPrecedings))
							for j := range keyTypes {
								currentPrecedings = append(currentPrecedings, keyTypes[j]...)
							}
							dm.innerlog.Log(log.LogLevel_Debug, "spawned txs  precedings ------------> ", zap.Int("precedings length", len(currentPrecedings)))
						}
						currentHashs := dm.sequenceLookup.Zoomin(convertHash(currentPrecedings))

						precedings := mergeHashList(dm.MainHashs, currentHashs)
						schedulingMsg.Precedings = &precedings
						schedulingMsg.PrecedingHash = common.CalculateHash(precedings)
					}
				} else {
					//
					precedings := dm.MainHashs
					precedingHash := common.CalculateHash(precedings)
					for _, msg := range sequence.Msgs {
						schedulingMsg, okk := msgs[msg.TxHash]
						if !okk {
							continue
						}
						schedulingMsg.Precedings = &precedings
						schedulingMsg.PrecedingHash = precedingHash
					}
				}
			}

		} else {
			//single txs
			if msg, ok := msgs[sequence.SequenceId]; ok {
				precedings := dm.MainHashs
				msg.Precedings = &precedings
				msg.PrecedingHash = common.CalculateHash(precedings)
			}
		}
	}
	go func(messages map[ethCommon.Hash]*schedulingTypes.Message) {
		dm.insertIntoGraph(messages)
	}(msgs)

	return false
}

func (dm *DependencyManager) OnBlockCompleted() {
	dm.workGraph = deepgraph.NewDeepgraph()
	dm.MainHashs = []*ethCommon.Hash{}
	dm.batchid = -1
	dm.removedTxs.clear()
	dm.spawnedTxs.clear()
	dm.executedTxs.clear()
	dm.sequenceLookup.Clear()
	dm.txidLookup.Clear()

	dm.spawnedRelations = []*types.SpawnedRelation{}
	dm.gasCache = schedulingTypes.NewGasCache()
}

func (dm *DependencyManager) ZoomIn(txElements [][][]*types.TxElement) [][][]*types.TxElement {
	results := make([][][]*types.TxElement, len(txElements))
	for _, rows := range txElements {
		list2 := make([][]*types.TxElement, 0, len(rows))
		for _, row := range rows {
			list := make([]*types.TxElement, 0, len(row))
			for _, col := range row {
				hashList := dm.sequenceLookup.Zoomin([]*ethCommon.Hash{col.TxHash})
				for _, hash := range hashList {
					list = append(list, &types.TxElement{
						TxHash:  hash,
						Batchid: col.Batchid,
					})
				}
			}
			list2 = append(list2, list)
		}
		results = append(results, list2)
	}
	return results
}

func (dm *DependencyManager) OnExecResultReceived(results map[ethCommon.Hash]*types.ExecuteResponse, spawnedHashes map[ethCommon.Hash]ethCommon.Hash, relations map[ethCommon.Hash][]ethCommon.Hash, broker *actor.MessageWrapper, inlog *actor.WorkerThreadLogger, msgs map[ethCommon.Hash]*schedulingTypes.Message, deftxids map[ethCommon.Hash]uint32, generationIdx, batchIdx int) ([]*schedulingTypes.Message, []uint32) {
	dm.txidLookup.Append(deftxids)
	dm.innerlog = inlog
	dm.innerlog.Log(log.LogLevel_Debug, "received result ", zap.Int("result length", len(results)))
	defs := map[string]*[]*types.ExecuteResponse{}

	for _, result := range results {
		dm.gasCache.Add(result.Hash, result.GasUsed)

		//gather arbitrate list
		if result.DfCall == nil || result.Status == ethTypes.ReceiptStatusFailed {
			continue
		}
		defid := result.DfCall.DeferID

		resultList := defs[defid]
		if resultList == nil {
			resultList = &[]*types.ExecuteResponse{}
			defs[defid] = resultList
		}
		*resultList = append(*resultList, result)
	}

	for hash, list := range relations {
		dm.sequenceLookup.Add(hash, list)
	}

	totalTxs := 0
	for _, sequence := range dm.sequences {
		totalTxs = totalTxs + len(sequence.Msgs)
	}

	hashlist := make([]*ethCommon.Hash, 0, totalTxs)
	for _, sequence := range dm.sequences {
		if sequence.Parallel {
			for _, msg := range sequence.Msgs {
				hashlist = append(hashlist, &msg.TxHash)
			}
		} else {
			hashes := dm.sequenceLookup.Zoomin([]*ethCommon.Hash{&sequence.SequenceId})
			hashlist = append(hashlist, hashes...)
		}
	}
	dm.executedTxs.addData(hashlist)

	arbitrateList := [][][]*types.TxElement{}
	if len(defs) == 0 {
		arbitrateList = dm.getArbitrateHashLatest()
	} else {
		arbitrateList = dm.getArbitrateHashs(&defs)
	}
	arbitrateList = dm.addArbitrateHash(arbitrateList)
	dm.innerlog.Log(log.LogLevel_Debug, "gather arbitrateList  completed", zap.Int("arbitrateList length", len(arbitrateList)))

	dm.gasCache.CostCalculateSort(&arbitrateList)
	dm.innerlog.Log(log.LogLevel_Debug, "CostCalculateSort  completed")
	arbitrateResults, cpairLeft, cpairRight := dm.arbitrator.Do(arbitrateList, inlog, generationIdx, batchIdx)
	dm.conflictlist.Add(cpairLeft, cpairRight)
	dm.innerlog.Log(log.LogLevel_Debug, "arbitrate  completed", zap.Int("arbitrateResults length", len(arbitrateResults)))

	willDeletes := make([]deepgraph.KeyType, 0, len(arbitrateResults))
	for _, hash := range arbitrateResults {
		willDeletes = append(willDeletes, deepgraph.KeyType(*hash))
	}

	willDeletes = dm.findDeleted(arbitrateList, willDeletes)
	dm.innerlog.Log(log.LogLevel_Debug, "find willDeletes", zap.Int("willDeletes length", len(willDeletes)))
	//delete from tree
	deleteKeys, deleteHashes := dm.getDeleteds(willDeletes)
	dm.workGraph.Remove(deleteKeys)

	dm.removedTxs.addData(deleteHashes)
	accumulatedRemovedTxs := dm.removedTxs.getData()

	spawnedTxs := dm.getSpawnedTxs(results, accumulatedRemovedTxs, msgs)
	txids := make([]uint32, 0, len(spawnedTxs))
	for _, spawnedTx := range spawnedTxs {
		if spawnedTx.DirectPrecedings == nil {
			continue
		}
		maxid := uint32(0)
		for _, txhash := range *spawnedTx.DirectPrecedings {
			dm.spawnedRelations = append(dm.spawnedRelations, &types.SpawnedRelation{
				Txhash:        *txhash,
				SpawnedTxHash: spawnedTx.Message.TxHash,
			})
			id := dm.txidLookup.GetId(*txhash)
			if id > maxid {
				maxid = id
			}
		}
		txids = append(txids, maxid)
	}

	dm.innerlog.Log(log.LogLevel_Debug, "gather spawnedTxs  completed", zap.Int("spawnedTxs length", len(spawnedTxs)))

	spawnedTxHashes := make([]*ethCommon.Hash, 0, len(spawnedTxs)+len(spawnedHashes))
	for _, tx := range spawnedTxs {
		dm.innerlog.Log(log.LogLevel_Debug, "tx.DirectPrecedings", zap.Int("DirectPrecedings length", len(*tx.DirectPrecedings)))
		spawnedTxHashes = append(spawnedTxHashes, &tx.Message.TxHash)
	}
	for _, hash := range spawnedHashes {
		spawnedTxHashes = append(spawnedTxHashes, &hash)
	}
	dm.spawnedTxs.addData(spawnedTxHashes)
	return spawnedTxs, schedulingTypes.GetSpawnedTxIds(txids)
}

func (dm *DependencyManager) getDeleteds(willdelete []deepgraph.KeyType) ([]deepgraph.KeyType, []*ethCommon.Hash) {
	deleteHashes := make([]*ethCommon.Hash, len(willdelete))
	deleteKeyTypes := map[deepgraph.KeyType]int{}
	for i, key := range willdelete {
		hash := ethCommon.Hash(key)
		deleteHashes[i] = &hash
		sequenceid := dm.sequenceLookup.Zoomout(hash)
		deleteKeyTypes[deepgraph.KeyType(*sequenceid)] = 0
	}
	deleteKeys := make([]deepgraph.KeyType, 0, len(deleteKeyTypes))
	for key, _ := range deleteKeyTypes {
		deleteKeys = append(deleteKeys, key)
	}
	return deleteKeys, deleteHashes
}

type Index struct {
	ki int
	kj int
}

func (dm *DependencyManager) findDeleted(arbitrateList [][][]*types.TxElement, willDeletes []deepgraph.KeyType) []deepgraph.KeyType {
	if len(arbitrateList) == 0 || len(willDeletes) == 0 {
		return willDeletes
	}
	index := map[ethCommon.Hash]*Index{}
	for ki := range arbitrateList {
		for kj := range arbitrateList[ki] {
			for j := range arbitrateList[ki][kj] {
				index[*arbitrateList[ki][kj][j].TxHash] = &Index{
					ki: ki,
					kj: kj,
				}
			}
		}
	}
	retdeletes := make([]deepgraph.KeyType, 0, len(willDeletes))
	for i := range willDeletes {
		if foundIndex, ok := index[ethCommon.Hash(willDeletes[i])]; ok {
			for j := range arbitrateList[foundIndex.ki][foundIndex.kj] {
				retdeletes = append(retdeletes, deepgraph.KeyType(*arbitrateList[foundIndex.ki][foundIndex.kj][j].TxHash))
			}
		}
	}
	return retdeletes
}

func (dm *DependencyManager) addArbitrateHash(txlist [][][]*types.TxElement) [][][]*types.TxElement {
	for _, txlst := range txlist {
		for _, lst := range txlst {
			for i := range lst {
				lst[i].Txid = dm.txidLookup.GetId(*lst[i].TxHash)
			}
		}
	}
	return txlist
}

func (dm *DependencyManager) getArbitrateHashLatest() [][][]*types.TxElement {
	keyTypes, batches, _ := dm.workGraph.GetSubgraphs()
	dm.innerlog.Log(log.LogLevel_Debug, "getArbitrateHashLatest", zap.Int("keyTypes length", len(keyTypes)))
	txlists := make([][]*types.TxElement, len(keyTypes))
	common.ParallelWorker(len(keyTypes), dm.concurrency, dm.txElementWorker, keyTypes, txlists, batches)
	if len(txlists) > 1 {
		return [][][]*types.TxElement{txlists}
	}
	return [][][]*types.TxElement{}
}

func (dm *DependencyManager) txElementWorker(start, end, idx int, args ...interface{}) {
	keyTypes := args[0].([]interface{})[0].([][]deepgraph.KeyType)
	txlist := args[0].([]interface{})[1].([][]*types.TxElement)
	batche := args[0].([]interface{})[2].([][]int)
	for i, rows := range keyTypes[start:end] {
		list := make([]*types.TxElement, 0, len(rows))
		idx := start + i
		for j, key := range rows {
			hash := ethCommon.Hash(key)
			largeList := dm.sequenceLookup.Zoomin([]*ethCommon.Hash{&hash})
			for _, nhash := range largeList {
				list = append(list, &types.TxElement{
					TxHash:  nhash,
					Batchid: uint64(batche[idx][j]),
				})
			}
		}
		txlist[idx] = list
	}
}

func (dm *DependencyManager) getArbitrateHashs(defs *map[string]*[]*types.ExecuteResponse) [][][]*types.TxElement {
	if defs == nil {
		return [][][]*types.TxElement{}
	}
	defTxlists := [][][]*types.TxElement{}
	for _, results := range *defs {
		if results == nil {
			continue
		}
		findList := make([]deepgraph.KeyType, 0, len(*results))
		for _, result := range *results {
			findList = append(findList, deepgraph.KeyType(result.Hash))
		}
		findResults, batches, _ := dm.workGraph.GetAncestorFamilies(findList)
		dm.innerlog.Log(log.LogLevel_Debug, "getArbitrateHashs", zap.Int("keyTypes length", len(findResults)), zap.Int("results", len(*results)))

		txlists := make([][]*types.TxElement, len(findResults))

		common.ParallelWorker(len(findResults), dm.concurrency, dm.txElementWorker, findResults, txlists, batches)
		if len(txlists) > 1 {
			defTxlists = append(defTxlists, txlists)
		}
	}
	return defTxlists
}

func (dm *DependencyManager) getSpawnedTxs(results map[ethCommon.Hash]*types.ExecuteResponse, removedTxs []*ethCommon.Hash, msgs map[ethCommon.Hash]*schedulingTypes.Message) []*schedulingTypes.Message {
	allRemovedList := map[ethCommon.Hash]int{}
	for i, hash := range removedTxs {
		allRemovedList[*hash] = i
	}

	defs := map[string][]*types.ExecuteResponse{}
	for _, result := range results {
		if _, ok := allRemovedList[result.Hash]; ok {
			continue
		}
		if result.DfCall == nil {
			continue
		}
		defid := result.DfCall.DeferID
		defs[defid] = append(defs[defid], result)
	}

	spawnedTxs := []*schedulingTypes.Message{}
	for _, results := range defs {
		hs := make([][]byte, len(results))
		for i, result := range results {
			hs[i] = result.Hash.Bytes()
		}
		sortedBytes, err := mhasher.SortBytes(hs)
		lastHash := ethCommon.BytesToHash(sortedBytes[len(sortedBytes)-1])
		nonce := msgs[lastHash].Message.Native.Nonce()
		if err != nil {
			continue
		}
		hash := sha256.Sum256(common.Flatten(sortedBytes))
		standardMessager := types.MakeMessageWithDefCall(results[0].DfCall, hash, nonce)
		directPrecedings := []*ethCommon.Hash{}
		for _, result := range results {
			directPrecedings = append(directPrecedings, &result.Hash)
		}
		msg := schedulingTypes.Message{
			Message:          standardMessager,
			IsSpawned:        true,
			DirectPrecedings: &directPrecedings,
		}
		spawnedTxs = append(spawnedTxs, &msg)
	}
	return spawnedTxs
}

func getInclusiveList(executed, removed []*ethCommon.Hash) *types.InclusiveList {
	removedDict := make(map[ethCommon.Hash]struct{})
	for _, v := range removed {
		removedDict[*v] = struct{}{}
	}
	listFlag := make([]bool, len(executed))
	for i, hash := range executed {
		_, ok := removedDict[*hash]
		listFlag[i] = !ok
	}

	return &types.InclusiveList{
		HashList:   executed,
		Successful: listFlag,
	}
}
