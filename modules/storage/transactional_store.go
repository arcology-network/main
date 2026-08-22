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
	"github.com/arcology-network/common-lib/storage/transactional"
	"github.com/arcology-network/streamer/actor"
)

type AddDataRequest struct {
	Data        interface{}
	RecoverFunc string
}

type TransactionalStore struct {
	tfdb         *transactional.TransactionalFileDB
	current      *transactional.Transaction
	previous     *transactional.Transaction
	optimization bool
}

func NewTransactionalStore() actor.Business {
	return &TransactionalStore{}
}

func (ts *TransactionalStore) Config(params map[string]interface{}) {
	ts.tfdb = transactional.NewTransactionalFileDB(params["root"].(string))
	ts.optimization = params["optimization"].(bool)
}

func (ts *TransactionalStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (ts *TransactionalStore) Outputs() map[string]int {
	return map[string]int{}
}

func (ts *TransactionalStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("BeginTransaction", ts.BeginTransaction)
	reg.Register("AddData", ts.AddData)
	reg.Register("EndTransaction", ts.EndTransaction)
	reg.Register("Recover", ts.Recover)
}

func (ts *TransactionalStore) RpcConfig() (string, int) {
	return "transactionalstore", 20
}

func (ts *TransactionalStore) BeginTransaction(ctx *actor.ActionContext) error {
	if ts.current != nil {
		panic("BeginTransaction called in another transaction.")
	}
	id := ctx.RPC.Request.(string)
	var err error
	ts.current, err = ts.tfdb.BeginTransaction(id)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
	} else {
		ctx.ExecCtx.SendRpcResponse("", nil)
	}

	return nil
}

func (ts *TransactionalStore) AddData(ctx *actor.ActionContext) error {
	if ts.current == nil {
		panic("AddData called before BeginTransaction.")
	}
	if ts.optimization {
		ctx.ExecCtx.SendRpcResponse("", 0)
	} else {
		request := ctx.RPC.Request.(*AddDataRequest)
		err := ts.current.Add(request.Data, request.RecoverFunc)
		if err != nil {
			ctx.ExecCtx.SendRpcResponse(err.Error(), 0)
		} else {
			ctx.ExecCtx.SendRpcResponse("", 0)
		}
	}
	return nil
}

func (ts *TransactionalStore) EndTransaction(ctx *actor.ActionContext) error {
	if ts.current == nil {
		panic("EndTransaction called before BeginTransaction.")
	}

	if ts.previous != nil {
		ts.previous.Clear()
	}
	err := ts.current.End()
	ts.previous = ts.current
	ts.current = nil
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), 0)
	} else {
		ctx.ExecCtx.SendRpcResponse("", 0)
	}
	return nil
}

func (ts *TransactionalStore) Recover(ctx *actor.ActionContext) error {
	id := ctx.RPC.Request.(string)
	err := ts.tfdb.Recover(id)
	if err != nil {
		ctx.ExecCtx.SendRpcResponse(err.Error(), 0)
	} else {
		ctx.ExecCtx.SendRpcResponse("", 0)
	}
	return nil
}
