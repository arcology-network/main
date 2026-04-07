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
	"github.com/arcology-network/common-lib/codec"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

type State struct {
	Height        uint64
	ParentHash    evmCommon.Hash
	ParentRoot    evmCommon.Hash
	ExcessBlobGas uint64
	BlobGasUsed   uint64
}

func (s *State) Encode() []byte {
	buffers := [][]byte{
		codec.Uint64(s.Height).Encode(),
		s.ParentHash.Bytes(),
		s.ParentRoot.Bytes(),
		codec.Uint64(s.ExcessBlobGas).Encode(),
		codec.Uint64(s.BlobGasUsed).Encode(),
	}
	return codec.Byteset(buffers).Encode()
}

func (s *State) Decode(data []byte) *State {
	buffers := [][]byte(codec.Byteset{}.Decode(data).(codec.Byteset))
	s.Height = uint64(codec.Uint64(0).Decode(buffers[0]).(codec.Uint64))
	s.ParentHash = evmCommon.BytesToHash(buffers[1])
	s.ParentRoot = evmCommon.BytesToHash(buffers[2])
	s.ExcessBlobGas = uint64(codec.Uint64(0).Decode(buffers[3]).(codec.Uint64))
	s.BlobGasUsed = uint64(codec.Uint64(0).Decode(buffers[4]).(codec.Uint64))
	return s
}

type StateStore struct {
	db    *mstypes.RawFile
	state *State
}

const (
	statefilename = "statestore"
)

func NewStateStore() actor.Business {
	return &StateStore{}
}

func (ss *StateStore) Config(params map[string]interface{}) {
	ss.db = mstypes.NewRawFiles(params["storage_state_path"].(string))
}

func (ss *StateStore) Inputs() ([]string, bool) {
	return []string{}, false
}

func (ss *StateStore) Outputs() map[string]int {
	return map[string]int{}
}

func (ss *StateStore) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register("Save", ss.Save)
	reg.Register("GetHeight", ss.GetHeight)
	reg.Register("GetParentInfo", ss.GetParentInfo)
}

func (ss *StateStore) RpcConfig() (string, int) {
	return "statestore", 20
}

func (ss *StateStore) Save(ctx *actor.ActionContext) error {
	request := ctx.RPC.Request.(*State)
	ss.state = request
	ss.db.Write(statefilename, ss.state.Encode())
	ctx.ExecCtx.SendRpcResponse("", nil)
	return nil
}

func (ss *StateStore) GetHeight(ctx *actor.ActionContext) error {
	if ss.state == nil {
		data, err := ss.db.Read(statefilename)
		if err != nil {
			ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
			return err
		}
		ss.state = &State{}
		ss.state = ss.state.Decode(data)
	}
	ctx.ExecCtx.SendRpcResponse("", ss.state.Height)
	return nil
}

func (ss *StateStore) GetParentInfo(ctx *actor.ActionContext) error {
	if ss.state == nil {
		data, err := ss.db.Read(statefilename)
		if err != nil {
			ctx.ExecCtx.SendRpcResponse(err.Error(), nil)
			return err
		}
		ss.state = ss.state.Decode(data)
	}
	ctx.ExecCtx.SendRpcResponse("", &mtypes.ParentInfo{
		ParentHash:    ss.state.ParentHash,
		ParentRoot:    ss.state.ParentRoot,
		ExcessBlobGas: ss.state.ExcessBlobGas,
		BlobGasUsed:   ss.state.BlobGasUsed,
	})
	return nil
}
