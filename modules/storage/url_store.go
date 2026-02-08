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
	strtyp "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	interfaces "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/streamer/actor"
)

const (
	ContainerTypeArray = iota
	ContainerTypeMap
	ContainerTypeQueue
)

type UrlStore struct {
	store   interfaces.ReadOnlyStore
	indexer *MetaIndexer
}

func NewUrlStore() actor.Business {
	return &UrlStore{
		indexer: NewMetaIndexer(),
	}
}

func (us *UrlStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (us *UrlStore) Outputs() map[string]int {
	return map[string]int{}
}

func (us *UrlStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Init", us.Init)
	reg.Register("Query", us.Query)
	reg.Register("Get", us.Get)
	reg.Register("GetNonce", us.GetNonce)
	reg.Register("GetBalance", us.GetBalance)
	reg.Register("GetCode", us.GetCode)
	reg.Register("GetEthStorage", us.GetEthStorage)
	reg.Register("ApplyData", us.ApplyData)
	reg.Register("RewriteMeta", us.RewriteMeta)
}

func (us *UrlStore) RpcConfig() (string, int) {
	return "urlstore", 20
}

func (us *UrlStore) Init(ctx *actor.ActionContext) error {
	store := ctx.RPC.Request.(interfaces.ReadOnlyStore)
	us.store = store
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (us *UrlStore) Query(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (us *UrlStore) Get(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (us *UrlStore) GetNonce(ctx *actor.ActionContext) error {
	address := ctx.RPC.Request.(string)
	nonce, err := strtyp.GetNonce(us.store, address)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nonce)
	}

	return nil
}

func (us *UrlStore) GetBalance(ctx *actor.ActionContext) error {
	address := ctx.RPC.Request.(string)
	balance, err := strtyp.GetBalance(us.store, address)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", balance)
	}

	return nil
}

func (us *UrlStore) GetCode(ctx *actor.ActionContext) error {
	address := ctx.RPC.Request.(string)
	code, err := strtyp.GetCode(us.store, address)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", code)
	}

	return nil
}

func (us *UrlStore) GetEthStorage(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*mtypes.UrlEthStorageGetRequest)
	value, err := strtyp.GetStorage(us.store, request.Address, request.Key)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", value)
	}

	return nil
}

func (us *UrlStore) ApplyData(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (us *UrlStore) RewriteMeta(ctx *actor.ActionContext) error {
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}
