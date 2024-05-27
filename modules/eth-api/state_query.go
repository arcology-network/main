/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package ethapi

import (
	"context"
	"fmt"
	"sync"

	apifunc "github.com/arcology-network/main/modules/eth-api/backend"
	mtypes "github.com/arcology-network/main/types"
	statestore "github.com/arcology-network/storage-committer"
	opadapter "github.com/arcology-network/storage-committer/op"
	ethdb "github.com/arcology-network/storage-committer/storage/ethstorage"
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
			ddb := v.Data.(*statestore.StateStore).Backend()
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
