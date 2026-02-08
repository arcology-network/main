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

package consensus

import (
	tmstate "github.com/arcology-network/consensus-engine/proto/tendermint/state"
	tmproto "github.com/arcology-network/consensus-engine/proto/tendermint/types"
	"github.com/arcology-network/consensus-engine/state"
	contyp "github.com/arcology-network/consensus-engine/types"
	"github.com/arcology-network/streamer/actor"
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
	sender  actor.OutboundSender
	from    string
}

func newStateStore(service string, sender actor.OutboundSender) state.Store {
	return &stateStore{
		service: service,
		sender:  sender,
		from:    "consensus",
	}
}

func (ss *stateStore) LoadFromDBOrGenesisFile(string) (state.State, error) {
	panic("not implemented")
}

func (ss *stateStore) LoadFromDBOrGenesisDoc(genesisDoc *contyp.GenesisDoc) (state.State, error) {
	state1, err := ss.sender.SendSync(ss.service, "LoadFromDBOrGenesisDoc", genesisDoc, 0, ss.from)
	if err != nil {
		return state.State{}, err
	}
	return state1.(state.State), nil
}

func (ss *stateStore) Load() (state.State, error) {
	state1, err := ss.sender.SendSync(ss.service, "Load", "", 0, ss.from)
	if err != nil {
		return state.State{}, err
	}
	return state1.(state.State), nil
}

func (ss *stateStore) LoadValidators(height int64) (*contyp.ValidatorSet, error) {
	vs, err := ss.sender.SendSync(ss.service, "LoadValidators", height, 0, ss.from)
	if err != nil {
		return nil, err
	}
	return vs.(*contyp.ValidatorSet), nil
}

func (ss *stateStore) LoadABCIResponses(height int64) (*tmstate.ABCIResponses, error) {
	responses, err := ss.sender.SendSync(ss.service, "LoadABCIResponses", height, 0, ss.from)
	if err != nil {
		return nil, err
	}
	return responses.(*tmstate.ABCIResponses), nil
}

func (ss *stateStore) LoadConsensusParams(height int64) (tmproto.ConsensusParams, error) {
	params, err := ss.sender.SendSync(ss.service, "LoadConsensusParams", height, 0, ss.from)
	if err != nil {
		return tmproto.ConsensusParams{}, err
	}
	return params.(tmproto.ConsensusParams), nil
}

func (ss *stateStore) Save(state state.State) error {
	_, err := ss.sender.SendSync(ss.service, "Save", &state, 0, ss.from)
	return err
}

func (ss *stateStore) SaveABCIResponses(height int64, responses *tmstate.ABCIResponses) error {
	_, err := ss.sender.SendSync(ss.service, "SaveABCIResponses", &SaveABCIResponsesRequest{
		Height:        height,
		ABCIResponses: responses,
	}, 0, ss.from)
	return err
}

func (ss *stateStore) Bootstrap(state state.State) error {
	_, err := ss.sender.SendSync(ss.service, "Bootstrap", &state, 0, ss.from)
	return err
}

func (ss *stateStore) PruneStates(from int64, to int64) error {
	_, err := ss.sender.SendSync(ss.service, "PruneStates", &PruneStatesRequest{
		From: from,
		To:   to,
	}, 0, ss.from)
	return err

}
