package ethapi

import (
	"context"
	"fmt"
	"sync"

	"github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/log"
	"github.com/arcology-network/concurrenturl/interfaces"
	ccdb "github.com/arcology-network/concurrenturl/storage"
	apifunc "github.com/arcology-network/main/modules/eth-api/backend"
	"go.uber.org/zap"
)

var (
	stateSingleton actor.IWorkerEx
	initStateOnce  sync.Once
)

type StateQuery struct {
	actor.WorkerThread
	store  *ccdb.EthDataStore
	manage *ccdb.MerkleProofManager
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
		// actor.MsgInitDB,
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
		case actor.MsgApcHandle: //actor.MsgInitDB: //
			ddb := (*v.Data.(*interfaces.Datastore)).(*ccdb.EthDataStore)
			// ddb := *v.Data.(*ccdb.EthDataStore)
			// proof, err := ccdb.NewMerkleProof(ddb.EthDB(), ddb.Root())
			// if err != nil {
			// 	panic("MerkleProof create failed")
			// }

			sq.store = ddb
			sq.manage = ccdb.NewMerkleProofManager(10, ddb.EthDB())
		}
	}
	return nil
}

func (sq *StateQuery) QueryState(ctx context.Context, request *types.QueryRequest, response *types.QueryResult) error {
	switch request.QueryType {
	case types.QueryType_Proof:
		rq := request.Data.(*types.RequestProof)
		keys := make([]string, len(rq.Keys))
		for i := range keys {
			keys[i] = fmt.Sprintf("%x", rq.Keys[i].Bytes())
		}
		sq.AddLog(log.LogLevel_Debug, "************* QueryState request", zap.String("blockTag", fmt.Sprintf("%x", rq.BlockTag)), zap.Strings("keys", keys), zap.String("addr", fmt.Sprintf("%x", rq.Address.Bytes())))

		rpcblock, err := apifunc.GetHeaderFromHash(rq.BlockTag)
		if err != nil {
			if err != nil {
				return err
			}
		}
		roothash := rpcblock.Header.Root.Bytes()
		// roothash := sq.store.Root()
		// proof, err := ccdb.NewMerkleProof(sq.store.EthDB(), [32]byte(roothash))
		// if err != nil {
		// 	return err
		// }
		// result, err := proof.GetProof(fmt.Sprintf("%x", rq.Address.Bytes()), keys)

		result, err := sq.manage.GetProof([32]byte(roothash), fmt.Sprintf("%x", rq.Address.Bytes()), keys)

		if err != nil {
			return err
		}
		response.Data = result
	}
	return nil
}
