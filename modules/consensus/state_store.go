package consensus

import (
	intf "github.com/arcology-network/component-lib/interface"
	tmstate "github.com/arcology-network/consensus-engine/proto/tendermint/state"
	tmproto "github.com/arcology-network/consensus-engine/proto/tendermint/types"
	"github.com/arcology-network/consensus-engine/state"
	contyp "github.com/arcology-network/consensus-engine/types"
)

type SaveABCIResponsesRequest struct {
	Height        int64
	ABCIResponses *tmstate.ABCIResponses
}

type PruneStatesRequest struct {
	From int64
	To   int64
}

type stateStore struct {
	service string
}

func newStateStore(service string) state.Store {
	return &stateStore{service: service}
}

func (ss *stateStore) LoadFromDBOrGenesisFile(string) (state.State, error) {
	panic("not implemented")
}

func (ss *stateStore) LoadFromDBOrGenesisDoc(genesisDoc *contyp.GenesisDoc) (state.State, error) {
	var state state.State
	err := intf.Router.Call(ss.service, "LoadFromDBOrGenesisDoc", genesisDoc, &state)
	return state, err
}

func (ss *stateStore) Load() (state.State, error) {
	var na int
	var state state.State
	err := intf.Router.Call(ss.service, "Load", &na, &state)
	return state, err
}

func (ss *stateStore) LoadValidators(height int64) (*contyp.ValidatorSet, error) {
	var vs *contyp.ValidatorSet
	err := intf.Router.Call(ss.service, "LoadValidators", &height, &vs)
	return vs, err
}

func (ss *stateStore) LoadABCIResponses(height int64) (*tmstate.ABCIResponses, error) {
	var responses *tmstate.ABCIResponses
	err := intf.Router.Call(ss.service, "LoadABCIResponses", &height, &responses)
	return responses, err
}

func (ss *stateStore) LoadConsensusParams(height int64) (tmproto.ConsensusParams, error) {
	var params tmproto.ConsensusParams
	err := intf.Router.Call(ss.service, "LoadConsensusParams", &height, &params)
	return params, err
}

func (ss *stateStore) Save(state state.State) error {
	var na int
	return intf.Router.Call(ss.service, "Save", &state, &na)
}

func (ss *stateStore) SaveABCIResponses(height int64, responses *tmstate.ABCIResponses) error {
	var na int
	return intf.Router.Call(ss.service, "SaveABCIResponses", &SaveABCIResponsesRequest{
		Height:        height,
		ABCIResponses: responses,
	}, &na)
}

func (ss *stateStore) Bootstrap(state state.State) error {
	var na int
	return intf.Router.Call(ss.service, "Bootstrap", &state, &na)
}

func (ss *stateStore) PruneStates(from int64, to int64) error {
	var na int
	return intf.Router.Call(ss.service, "PruneStates", &PruneStatesRequest{
		From: from,
		To:   to,
	}, &na)
}
