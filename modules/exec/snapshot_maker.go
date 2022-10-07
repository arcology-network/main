package exec

import (
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/storage"
	ccurl "github.com/HPISTechnologies/concurrenturl/v2"
	"go.uber.org/zap"

	execTypes "github.com/HPISTechnologies/main/modules/exec/types"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/component-lib/log"
	urlcommon "github.com/HPISTechnologies/concurrenturl/v2/common"

	curstorage "github.com/HPISTechnologies/concurrenturl/v2/storage"

	urltyp "github.com/HPISTechnologies/concurrenturl/v2/type"
)

type SnapshotDict interface {
	Reset(snapshot *urlcommon.DatastoreInterface)
	AddItem(precedingHash ethCommon.Hash, size int, snapshot *urlcommon.DatastoreInterface)
	Query(precedings []*ethCommon.Hash) (*urlcommon.DatastoreInterface, []*ethCommon.Hash)
}

var (
	SnapshotLookup SnapshotDict = execTypes.NewLookup()
)

type SnapshotMaker struct {
	actor.WorkerThread
	txs         execTypes.TxToExecutes
	apcBlock    *execTypes.ApcBlock
	paramsCache map[uint64]*types.ExecutorRequest

	lookup        SnapshotDict
	snapshot      *urlcommon.DatastoreInterface
	precedingHash ethCommon.Hash
	precedingSize int
	url           *ccurl.ConcurrentUrl
	height        uint64
}

//return a Subscriber struct
func NewSnapshotMaker(
	concurrency int,
	groupid string,
) actor.IWorkerEx {
	sm := SnapshotMaker{}
	sm.Set(concurrency, groupid)
	return &sm
}

func (sm *SnapshotMaker) Inputs() ([]string, bool) {
	return []string{
		//actor.MsgApcHandle,
		actor.CombinedName(actor.MsgApcHandle, actor.MsgCached),
		actor.MsgPrecedingsEuresult,
		actor.MsgTxs,
	}, false
}

func (sm *SnapshotMaker) Outputs() map[string]int {
	return map[string]int{actor.MsgPrecedingList: 1, actor.MsgTxsToExecute: 1}
}

func (sm *SnapshotMaker) OnStart() {
	sm.paramsCache = map[uint64]*types.ExecutorRequest{}
	sm.apcBlock = &execTypes.ApcBlock{
		ApcHeight: 0,
	}
	sm.url = ccurl.NewConcurrentUrl(nil)
}

func (sm *SnapshotMaker) Stop() {

}

func (sm *SnapshotMaker) createFirstSnapShot() {
	SnapshotLookup.Reset(sm.apcBlock.DB)
}

func (sm *SnapshotMaker) execResult(results []*types.EuResult) {

	if results == nil {
		return
	}

	db := curstorage.NewTransientDB(*sm.snapshot)
	sm.url.Init(db)
	txIds := storage.GetTransitionIds(results)

	for i := range results {
		univalues := urltyp.Univalues{}
		sm.url.Import(univalues.DecodeV2(results[i].Transitions, func() interface{} { return &urltyp.Univalue{} }, nil), false)
	}

	sm.url.PostImport()
	sm.url.Commit(txIds)
	SnapshotLookup.AddItem(sm.precedingHash, sm.precedingSize, &db)

	sm.txs.SnapShots = &db
	sm.MsgBroker.Send(actor.MsgTxsToExecute, &sm.txs, sm.height)
}

func (sm *SnapshotMaker) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgPrecedingsEuresult:
			datas := v.Data.([]interface{})
			results := make([]*types.EuResult, len(datas))
			for i, data := range datas {
				results[i] = data.(*types.EuResult)
			}
			if results != nil {
				sm.execResult(results)
			}
		case actor.MsgTxs:
			params := v.Data.(*types.ExecutorRequest)
			sm.height = v.Height
			if v.Height == sm.apcBlock.ApcHeight {
				sm.AddLog(log.LogLevel_Debug, "exec ExecutorRequest", zap.Uint64("apcHeight", sm.apcBlock.ApcHeight), zap.Uint64("ExecutorRequest height", v.Height))
				sm.execRequest(params)
			} else {
				sm.paramsCache[v.Height] = params
				sm.AddLog(log.LogLevel_Debug, "cache ExecutorRequest", zap.Uint64("apcHeight", sm.apcBlock.ApcHeight), zap.Uint64("ExecutorRequest height", v.Height))
			}
		case actor.CombinedName(actor.MsgApcHandle, actor.MsgCached):
			//sm.apcBlock.DB = v.Data.(*urlcommon.DatastoreInterface)
			combined := v.Data.(*actor.CombinerElements)
			sm.apcBlock.DB = combined.Get(actor.MsgApcHandle).Data.(*urlcommon.DatastoreInterface)

			sm.apcBlock.ApcHeight = v.Height + 1
			sm.createFirstSnapShot()
			if params, ok := sm.paramsCache[sm.apcBlock.ApcHeight]; ok {
				sm.AddLog(log.LogLevel_Debug, "exec MsgApcHandle", zap.Uint64("apcHeight", sm.apcBlock.ApcHeight))
				sm.execRequest(params)
				sm.paramsCache = map[uint64]*types.ExecutorRequest{}
			}
		}
	}
	return nil
}

func (sm *SnapshotMaker) execRequest(params *types.ExecutorRequest) {
	if params != nil {
		sm.txs = execTypes.TxToExecutes{
			Sequences:   params.Sequences,
			Timestamp:   params.Timestamp,
			Parallelism: params.Parallelism,
			Debug:       params.Debug,
		}

		sm.txs.PrecedingHash = params.PrecedingHash
		snapshot, precedings := SnapshotLookup.Query(params.Precedings)

		sm.snapshot = snapshot
		sm.precedingHash = params.PrecedingHash
		sm.precedingSize = len(params.Precedings)
		if len(precedings) == 0 {
			sm.txs.SnapShots = snapshot
			sm.MsgBroker.Send(actor.MsgTxsToExecute, &sm.txs, sm.height)
		} else {
			sm.MsgBroker.Send(actor.MsgPrecedingList, &precedings, sm.height)
		}

	}
}
