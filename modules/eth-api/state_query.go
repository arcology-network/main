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
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	statestore "github.com/arcology-network/state-engine"
	opadapter "github.com/arcology-network/state-engine/op"
	ethstg "github.com/arcology-network/state-engine/storage/ethstorage"
	"github.com/arcology-network/state-engine/storage/proxy"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type StateQuery struct {
	ProofCache *ethstg.MerkleProofCache
	request    *mtypes.RequestProof
}

// return a Subscriber struct
func NewStateQuery() actor.Business {

	return &StateQuery{}

}

func (sq *StateQuery) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgApcHandle,
	}, false
}

func (sq *StateQuery) Outputs() map[string]int {
	return map[string]int{}
}

func (sq *StateQuery) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgApcHandle, sq.updateApchandle)
	reg.Register("QueryState", sq.QueryState)
	reg.Register("onBlockQueried", sq.onBlockQueried)
}

func (sq *StateQuery) updateApchandle(ctx *actor.ActionContext) error {
	ddb := ctx.Messages[0].Data.(*statestore.StateStore)
	cache := ethstg.NewMerkleProofCache(2, ddb.ReadOnlyStore().(*proxy.StorageProxy).EthStore().EthDB())
	sq.ProofCache = cache

	return nil
}
func (sq *StateQuery) RpcConfig() (string, int) {
	return "state_query", 20
}

func (sq *StateQuery) onBlockQueried(ctx *actor.ActionContext) error {
	rpcblock := ctx.RPC.Request.(*mtypes.QueryResult).Data.(*mtypes.RPCBlock)
	var err error
	roothash := rpcblock.Header.Root

	// Get the proof provider by a root hash.
	provider, err := sq.ProofCache.GetProofProvider(roothash)
	if err != nil {
		panic(err)
	}

	keys := make([]string, len(sq.request.Keys))
	for i := range keys {
		keys[i] = fmt.Sprintf("%x", sq.request.Keys[i].Bytes())
	}

	ctx.ExecCtx.LogDebug("QueryState request", logger.F("keys", keys), logger.F("addr", fmt.Sprintf("%x", sq.request.Address.Bytes())))

	accountResult, err := provider.GetProof(sq.request.Address, keys)
	if err := accountResult.Validate(roothash); err != nil {
		ctx.ExecCtx.LogErr("accountResult Validate Failed", logger.F("err", err))
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return err
	}

	// Convert to OP format and verify.
	opProof := opadapter.Convertible(*accountResult).New() // To OP format
	if err := opProof.Verify(roothash); err != nil {
		ctx.ExecCtx.LogErr("accountResult Convert Failed", logger.F("err", err))
	}

	ctx.ExecCtx.SendRpcResponse("", accountResult)
	// }
	return nil
}

func (sq *StateQuery) QueryState(ctx *actor.ActionContext) error {
	sq.request = ctx.RPC.Request.(*mtypes.RequestProof)

	if hash, ok := sq.request.BlockParameter.Hash(); ok {
		ctx.ExecCtx.InvokeRPC("storage", "Query", &mtypes.QueryRequest{
			QueryType: mtypes.QueryType_HeaderByHash,
			Data: &mtypes.RequestBlockEth{
				Hash: hash,
			},
		}, "onBlockQueried")
	} else if number, ok := sq.request.BlockParameter.Number(); ok {
		ctx.ExecCtx.InvokeRPC("storage", "Query", &mtypes.QueryRequest{
			QueryType: mtypes.QueryType_HeaderByNumber,
			Data: &mtypes.RequestBlockEth{
				Number: number.Int64(),
			},
		}, "onBlockQueried")
	} else {
		ctx.ExecCtx.SendRpcResponse("query err", nil)
	}

	return nil
}
