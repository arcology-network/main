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

package storage

import (
	"github.com/arcology-network/consensus-engine/state"
	contyp "github.com/arcology-network/consensus-engine/types"
	conwrk "github.com/arcology-network/main/modules/consensus"
	"github.com/arcology-network/streamer/actor"
	tmdb "github.com/tendermint/tm-db"
)

type TmStateStore struct {
	impl state.Store
}

func NewTmStateStore() actor.Business {
	return &TmStateStore{}
}

func (tss *TmStateStore) Config(params map[string]interface{}) {
	db, err := tmdb.NewDB("tm_state_store", tmdb.GoLevelDBBackend, params["tm_state_store_dir"].(string))
	if err != nil {
		panic(err)
	}

	tss.impl = state.NewStore(db)
}

func (tss *TmStateStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (tss *TmStateStore) Outputs() map[string]int {
	return map[string]int{}
}

func (tss *TmStateStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Load", tss.Load)
	reg.Register("LoadValidators", tss.LoadValidators)
	reg.Register("LoadABCIResponses", tss.LoadABCIResponses)
	reg.Register("LoadConsensusParams", tss.LoadConsensusParams)
	reg.Register("Save", tss.Save)
	reg.Register("SaveABCIResponses", tss.SaveABCIResponses)
	reg.Register("Bootstrap", tss.Bootstrap)
	reg.Register("PruneStates", tss.PruneStates)
	reg.Register("LoadFromDBOrGenesisDoc", tss.LoadFromDBOrGenesisDoc)
}

func (tss *TmStateStore) RpcConfig() (string, int) {
	return "tmstatestore", 20
}

func (tss *TmStateStore) Load(ctx *actor.ActionContext) error {
	state, err := tss.impl.Load()
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", state)
	}
	return nil
}

func (tss *TmStateStore) LoadValidators(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(int64)
	vs, err := tss.impl.LoadValidators(height)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", vs)
	}
	return nil
}

func (tss *TmStateStore) LoadABCIResponses(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(int64)
	responses, err := tss.impl.LoadABCIResponses(height)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", responses)
	}
	return nil
}

func (tss *TmStateStore) LoadConsensusParams(ctx *actor.ActionContext) error {
	height := ctx.RPC.Request.(int64)
	params, err := tss.impl.LoadConsensusParams(height)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", params)
	}
	return nil
}

func (tss *TmStateStore) Save(ctx *actor.ActionContext) error {
	state := ctx.RPC.Request.(*state.State)
	err := tss.impl.Save(*state)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nil)
	}
	return nil
}

func (tss *TmStateStore) SaveABCIResponses(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*conwrk.SaveABCIResponsesRequest)
	err := tss.impl.SaveABCIResponses(request.Height, request.ABCIResponses)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nil)
	}
	return nil
}

func (tss *TmStateStore) Bootstrap(ctx *actor.ActionContext) error {
	state := ctx.RPC.Request.(*state.State)
	err := tss.impl.Bootstrap(*state)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nil)
	}
	return nil
}

func (tss *TmStateStore) PruneStates(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*conwrk.PruneStatesRequest)
	err := tss.impl.PruneStates(request.From, request.To)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nil)
	}
	return nil
}

func (tss *TmStateStore) LoadFromDBOrGenesisDoc(ctx *actor.ActionContext) error {
	genesisDoc := ctx.RPC.Request.(*contyp.GenesisDoc)
	state, err := tss.impl.LoadFromDBOrGenesisDoc(genesisDoc)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", state)
	}
	return nil
}
