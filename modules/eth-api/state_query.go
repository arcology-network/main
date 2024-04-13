package ethapi

import (
	"context"
	"fmt"
	"sync"

	apifunc "github.com/arcology-network/main/modules/eth-api/backend"
	mtypes "github.com/arcology-network/main/types"
	opadapter "github.com/arcology-network/storage-committer/op"
	ethdb "github.com/arcology-network/storage-committer/storage/ethstorage"
	stgproxy "github.com/arcology-network/storage-committer/storage/proxy"
	"github.com/arcology-network/storage-committer/storage/statestore"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	"go.uber.org/zap"
)

var (
	stateSingleton actor.IWorkerEx
	initStateOnce  sync.Once
)

type StateQuery struct {
	actor.WorkerThread
	ProofCache *ethdb.MerkleProofCache
}

// return a Subscriber struct
func NewStateQuery(concurrency int, groupid string) actor.IWorkerEx {
	initStateOnce.Do(func() {
		stateSingleton = &StateQuery{}
		stateSingleton.(*StateQuery).Set(concurrency, groupid)
	})
	return stateSingleton
}

func (sq *StateQuery) Inputs() ([]string, bool) {
	return []string{
		actor.MsgApcHandle,
	}, true
}

func (sq *StateQuery) Outputs() map[string]int {
	return map[string]int{}
}

func (sq *StateQuery) Config(params map[string]interface{}) {

}

func (*StateQuery) OnStart() {

}

func (*StateQuery) Stop() {}

func (sq *StateQuery) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgApcHandle:
			ddb := v.Data.(*statestore.StateStore).Store().(*stgproxy.StorageProxy)
			cache := ethdb.NewMerkleProofCache(2, ddb.EthStore().EthDB())
			sq.ProofCache = cache
		}
	}
	return nil
}

func (sq *StateQuery) QueryState(ctx context.Context, request *mtypes.QueryRequest, response *mtypes.QueryResult) error {
	switch request.QueryType {
	case mtypes.QueryType_Proof:
		rq := request.Data.(*mtypes.RequestProof)
		keys := make([]string, len(rq.Keys))
		for i := range keys {
			keys[i] = fmt.Sprintf("%x", rq.Keys[i].Bytes())
		}
		sq.AddLog(log.LogLevel_Debug, "************* QueryState request", zap.Strings("keys", keys), zap.String("addr", fmt.Sprintf("%x", rq.Address.Bytes())))
		var rpcblock *mtypes.RPCBlock
		var err error
		if hash, ok := rq.BlockParameter.Hash(); ok {
			rpcblock, err = apifunc.GetHeaderFromHash(hash)
		} else if number, ok := rq.BlockParameter.Number(); ok {
			rpcblock, err = apifunc.GetHeaderByNumber(number.Int64())
		} else {
			return err
		}

		if err != nil {
			if err != nil {
				return err
			}
		}
		roothash := rpcblock.Header.Root

		// Get the proof provider by a root hash.
		provider, err := sq.ProofCache.GetProofProvider(roothash)
		if err != nil {
			panic(err)
		}

		accountResult, err := provider.GetProof(rq.Address, keys)
		if err := accountResult.Validate(roothash); err != nil {
			sq.AddLog(log.LogLevel_Error, "accountResult Validate Failed", zap.Error(err))
			return err
		}

		// Convert to OP format and verify.
		opProof := opadapter.Convertible(*accountResult).New() // To OP format
		if err := opProof.Verify(roothash); err != nil {
			sq.AddLog(log.LogLevel_Error, "accountResult Convert Failed", zap.Error(err))
		}

		response.Data = accountResult
	}
	return nil
}
