package consensus

import (
	abci "github.com/HPISTechnologies/consensus-engine/abci/types"
)

//	txpool node
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
