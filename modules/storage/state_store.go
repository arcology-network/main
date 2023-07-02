package storage

import (
	"context"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/types"
	evmCommon "github.com/arcology-network/evm/common"
	mstypes "github.com/arcology-network/main/modules/storage/types"
)

type State struct {
	Height     uint64
	ParentHash evmCommon.Hash
	ParentRoot evmCommon.Hash
}

func (s *State) Encode() []byte {
	buffers := [][]byte{
		codec.Uint64(s.Height).Encode(),
		s.ParentHash.Bytes(),
		s.ParentRoot.Bytes(),
	}
	return codec.Byteset(buffers).Encode()
}

func (s *State) Decode(data []byte) *State {
	buffers := [][]byte(codec.Byteset{}.Decode(data).(codec.Byteset))
	s.Height = uint64(codec.Uint64(0).Decode(buffers[0]).(codec.Uint64))
	s.ParentHash = evmCommon.BytesToHash(buffers[1])
	s.ParentRoot = evmCommon.BytesToHash(buffers[2])
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

func (ss *StateStore) GetParentInfo(ctx context.Context, na *int, parentInfo *types.ParentInfo) error {
	if ss.state == nil {
		data, err := ss.db.Read(statefilename)
		if err != nil {
			return err
		}
		ss.state = ss.state.Decode(data)
	}
	parentInfo.ParentHash = ss.state.ParentHash
	parentInfo.ParentRoot = ss.state.ParentRoot
	return nil
}
