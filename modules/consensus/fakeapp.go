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
	abci "github.com/arcology-network/consensus-engine/abci/types"
)

// txpool node
type FakeApp struct {
}

// Info/Query Connection
// ABCI::Info
func (app *FakeApp) Info(req abci.RequestInfo) abci.ResponseInfo {
	return abci.ResponseInfo{}
}

// ABCI::SetOption
func (app *FakeApp) SetOption(req abci.RequestSetOption) (res abci.ResponseSetOption) {
	return
}

// ABCI::Query
func (app *FakeApp) Query(reqQuery abci.RequestQuery) (res abci.ResponseQuery) {
	return
}

// Mempool Connection
// ABCI::CheckTx
func (app *FakeApp) CheckTx(req abci.RequestCheckTx) (res abci.ResponseCheckTx) {
	return
}

// Consensus Connection
// ABCI::DeliverTx
func (app *FakeApp) DeliverTx(req abci.RequestDeliverTx) (res abci.ResponseDeliverTx) {
	//app.logger.Info("DeliverTx() enter")
	return
}

// ABCI::InitChain
func (app *FakeApp) InitChain(req abci.RequestInitChain) (res abci.ResponseInitChain) {
	return
}

// ABCI::BeginBlock
func (app *FakeApp) BeginBlock(req abci.RequestBeginBlock) (res abci.ResponseBeginBlock) {
	return
}

// ABCI::EndBlock
func (app *FakeApp) EndBlock(req abci.RequestEndBlock) (res abci.ResponseEndBlock) {
	return
}

// ABCI::Commit
func (app *FakeApp) Commit() (res abci.ResponseCommit) {
	return
}

// List available snapshots
func (app *FakeApp) ListSnapshots(req abci.RequestListSnapshots) (res abci.ResponseListSnapshots) {
	return
}

// Offer a snapshot to the application
func (app *FakeApp) OfferSnapshot(req abci.RequestOfferSnapshot) (res abci.ResponseOfferSnapshot) {
	return
}

// Load a snapshot chunk
func (app *FakeApp) LoadSnapshotChunk(req abci.RequestLoadSnapshotChunk) (res abci.ResponseLoadSnapshotChunk) {
	return
}

// Apply a shapshot chunk
func (app *FakeApp) ApplySnapshotChunk(req abci.RequestApplySnapshotChunk) (res abci.ResponseApplySnapshotChunk) {
	return
}
