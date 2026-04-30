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
	"math/big"

	mtypes "github.com/arcology-network/main/types"
	statestore "github.com/arcology-network/state-engine"
	proxy "github.com/arcology-network/state-engine/storage/proxy"
	"github.com/arcology-network/streamer/actor"
	"github.com/ethereum/go-ethereum/crypto"
	ethmpt "github.com/ethereum/go-ethereum/trie"
)

const (
	ContainerTypeArray = iota
	ContainerTypeMap
	ContainerTypeQueue
)

type UrlStore struct {
	stateStore *statestore.StateStore
}

func NewUrlStore() actor.Business {
	return &UrlStore{
		// indexer: NewMetaIndexer(),
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
	store := ctx.RPC.Request.(*statestore.StateStore)
	us.stateStore = store
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
	queryParam := ctx.RPC.Request.(*mtypes.QueryBlockParam)
	snapshot, err := proxy.NewEthStateSnapshot([32]byte(queryParam.Root), us.stateStore.ReadOnlyStore().(*proxy.StorageProxy).EthStore().Backend())
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return nil
	}

	account, err := snapshot.GetAccount(queryParam.Address, &ethmpt.AccessListCache{})
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", account.Nonce)
	}

	return nil
}

func (us *UrlStore) GetBalance(ctx *actor.ActionContext) error {
	queryParam := ctx.RPC.Request.(*mtypes.QueryBlockParam)

	snapshot, err := proxy.NewEthStateSnapshot([32]byte(queryParam.Root), us.stateStore.ReadOnlyStore().(*proxy.StorageProxy).EthStore().Backend())
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return nil
	}

	account, err := snapshot.GetAccount(queryParam.Address, &ethmpt.AccessListCache{})
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		bal := big.NewInt(0)
		if account != nil {
			bal = account.Balance.ToBig()
		}
		ctx.ExecCtx.SendRpcResponse("", bal)
	}

	return nil
}

func (us *UrlStore) GetCode(ctx *actor.ActionContext) error {
	queryParam := ctx.RPC.Request.(*mtypes.QueryBlockParam)
	snapshot, err := proxy.NewEthStateSnapshot([32]byte(queryParam.Root), us.stateStore.ReadOnlyStore().(*proxy.StorageProxy).EthStore().Backend())
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return nil
	}

	account, err := snapshot.GetAccount(queryParam.Address, &ethmpt.AccessListCache{})
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		if account == nil {
			ctx.ExecCtx.SendRpcResponse("", []byte{})
		} else {
			ctx.ExecCtx.SendRpcResponse("", account.GetCode())
		}

	}

	return nil
}

func (us *UrlStore) GetEthStorage(ctx *actor.ActionContext) error {
	queryParam := ctx.RPC.Request.(*mtypes.QueryBlockParam)
	backend := us.stateStore.ReadOnlyStore().(*proxy.StorageProxy).EthStore().Backend()
	snapshot, err := proxy.NewEthStateSnapshot([32]byte(queryParam.Root), backend)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
		return nil
	}

	account, err := snapshot.GetAccount(queryParam.Address, &ethmpt.AccessListCache{})
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		if account == nil {
			ctx.ExecCtx.SendRpcResponse("", []byte{})
		} else {
			ctx.ExecCtx.SendRpcResponse("", account.GetState([32]byte(crypto.Keccak256(queryParam.Key))))
		}
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
