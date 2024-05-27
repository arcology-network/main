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

	tmstate "github.com/arcology-network/consensus-engine/proto/tendermint/state"
	tmproto "github.com/arcology-network/consensus-engine/proto/tendermint/types"
	"github.com/arcology-network/consensus-engine/state"
	contyp "github.com/arcology-network/consensus-engine/types"
	conwrk "github.com/arcology-network/main/modules/consensus"
	tmdb "github.com/tendermint/tm-db"
)

type TmStateStore struct {
	impl state.Store
}

func NewTmStateStore() *TmStateStore {
	return &TmStateStore{}
}

func (tss *TmStateStore) Config(params map[string]interface{}) {
	db, err := tmdb.NewDB("tm_state_store", tmdb.GoLevelDBBackend, params["tm_state_store_dir"].(string))
	if err != nil {
		panic(err)
	}

	tss.impl = state.NewStore(db)
}

func (tss *TmStateStore) Load(ctx context.Context, _ *int, state *state.State) (err error) {
	*state, err = tss.impl.Load()
	return
}

func (tss *TmStateStore) LoadValidators(ctx context.Context, height *int64, vs **contyp.ValidatorSet) (err error) {
	*vs, err = tss.impl.LoadValidators(*height)
	return
}

func (tss *TmStateStore) LoadABCIResponses(ctx context.Context, height *int64, responses **tmstate.ABCIResponses) (err error) {
	*responses, err = tss.impl.LoadABCIResponses(*height)
	return
}

func (tss *TmStateStore) LoadConsensusParams(ctx context.Context, height *int64, params *tmproto.ConsensusParams) (err error) {
	*params, err = tss.impl.LoadConsensusParams(*height)
	return
}

func (tss *TmStateStore) Save(ctx context.Context, state *state.State, _ *int) error {
	return tss.impl.Save(*state)
}

func (tss *TmStateStore) SaveABCIResponses(ctx context.Context, request *conwrk.SaveABCIResponsesRequest, _ *int) error {
	return tss.impl.SaveABCIResponses(request.Height, request.ABCIResponses)
}

func (tss *TmStateStore) Bootstrap(ctx context.Context, state *state.State, _ *int) error {
	return tss.impl.Bootstrap(*state)
}

func (tss *TmStateStore) PruneStates(ctx context.Context, request *conwrk.PruneStatesRequest, _ *int) error {
	return tss.impl.PruneStates(request.From, request.To)
}

func (tss *TmStateStore) LoadFromDBOrGenesisDoc(ctx context.Context, genesisDoc *contyp.GenesisDoc, state *state.State) (err error) {
	*state, err = tss.impl.LoadFromDBOrGenesisDoc(genesisDoc)
	return
}
