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
	"context"

	"github.com/arcology-network/common-lib/codec"
	mstypes "github.com/arcology-network/main/modules/storage/types"
	mtypes "github.com/arcology-network/main/types"
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
	s.ExcessBlobGas = uint64(codec.Uint64(0).Decode(buffers[2]).(codec.Uint64))
	s.BlobGasUsed = uint64(codec.Uint64(0).Decode(buffers[3]).(codec.Uint64))
	return s
}

type StateStore struct {
	db    *mstypes.RawFile
	state *State
}

const (
	statefilename = "statestore"
)

func NewStateStore() *StateStore {
	return &StateStore{
		// TODO
	}
}

func (ss *StateStore) Config(params map[string]interface{}) {
	ss.db = mstypes.NewRawFiles(params["storage_state_path"].(string))
}

func (ss *StateStore) Save(ctx context.Context, request *State, _ *int) error {
	ss.state = request
	ss.db.Write(statefilename, ss.state.Encode())
	return nil
}

func (ss *StateStore) GetHeight(ctx context.Context, _ *int, height *uint64) error {
	if ss.state == nil {
		data, err := ss.db.Read(statefilename)
		if err != nil {
			return err
		}
		ss.state = &State{}
		ss.state = ss.state.Decode(data)
	}
	*height = ss.state.Height
	return nil
}

func (ss *StateStore) GetParentInfo(ctx context.Context, na *int, parentInfo *mtypes.ParentInfo) error {
	if ss.state == nil {
		data, err := ss.db.Read(statefilename)
		if err != nil {
			return err
		}
		ss.state = ss.state.Decode(data)
	}
	parentInfo.ParentHash = ss.state.ParentHash
	parentInfo.ParentRoot = ss.state.ParentRoot
	parentInfo.ExcessBlobGas = ss.state.ExcessBlobGas
	parentInfo.BlobGasUsed = ss.state.BlobGasUsed
	return nil
}
