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

package coordinator

import (
	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

const (
	dmStateUninit = iota
	dmStateFastSync
	dmStateBlockSyncWaiting
	dmStateBlockSync
)

type DecisionMaker struct {
	state                int
	fastSyncUntil        uint64
	fastSyncMessageTypes map[string]struct{}
	updateFSM            bool
	storageUp            bool
	maxPeerHeights       []uint64
	consensusUp          bool
	initHeight           uint64

	genesisData *actor.BlockStart
}

func NewDecisionMaker() actor.Business {
	dm := &DecisionMaker{
		state: dmStateUninit,
		fastSyncMessageTypes: map[string]struct{}{
			scommon.MsgConsensusMaxPeerHeight: {},
			scommon.MsgConsensusUp:            {},
			scommon.MsgExtBlockStart:          {},
			scommon.MsgExtBlockEnd:            {},
			scommon.MsgExtReapCommand:         {},
			scommon.MsgExtTxBlocks:            {},
			scommon.MsgExtReapingList:         {},
			scommon.MsgExtBlockCompleted:      {},
		},
	}
	return dm
}

func (dm *DecisionMaker) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgInitialization,
		scommon.MsgConsensusMaxPeerHeight,
		scommon.MsgConsensusUp,
		scommon.MsgExtBlockStart,
		scommon.MsgExtBlockEnd,
		scommon.MsgExtReapCommand,
		scommon.MsgExtTxBlocks,
		scommon.MsgExtBlockCompleted,
		scommon.MsgExtReapingList,
		scommon.MsgAppHash,
		scommon.MsgStateSyncDone,
	}, false
}

func (dm *DecisionMaker) Outputs() map[string]int {
	return map[string]int{
		scommon.MsgBlockStart:     1,
		scommon.MsgBlockEnd:       1,
		scommon.MsgReapCommand:    1,
		scommon.MsgTxBlocks:       1,
		scommon.MsgBlockCompleted: 1,
		scommon.MsgReapinglist:    1,
		scommon.MsgMetaBlock:      1,
		scommon.MsgExtAppHash:     1,
		scommon.MsgStateSyncStart: 1,
	}
}

func (dm *DecisionMaker) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitialization, dm.receivedInitialization)
	reg.Register(scommon.MsgConsensusMaxPeerHeight, dm.receivedConsensusMaxPeerHeight)
	reg.Register(scommon.MsgConsensusUp, dm.receivedConsensusUp)
	reg.Register(scommon.MsgExtBlockStart, dm.receivedExtBlockStart)
	reg.Register(scommon.MsgExtReapCommand, dm.receivedExtReapCommand)
	reg.Register(scommon.MsgExtReapingList, dm.receivedExtReapingList)
	reg.Register(scommon.MsgExtBlockEnd, dm.receivedExtBlockEnd)
	reg.Register(scommon.MsgExtBlockCompleted, dm.receivedExtBlockCompleted)
	reg.Register(scommon.MsgStateSyncDone, dm.receivedStateSyncDone)
	reg.Register(scommon.MsgExtTxBlocks, dm.receivedExtTxBlocks)
	reg.Register(scommon.MsgAppHash, dm.receivedAppHash)

}

func (dm *DecisionMaker) receivedInitialization(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	dm.storageUp = true
	dm.initHeight = msg.Height
	initialization := msg.Data.(*mtypes.Initialization)
	dm.genesisData = initialization.BlockStart
	dm.changeStateFromInit(ctx)
	return nil
}

func (dm *DecisionMaker) changeStateFromInit(ctx *actor.ActionContext) {
	if dm.storageUp && dm.consensusUp {
		ctx.ExecCtx.Send(scommon.MsgReapCommand, "")
		dm.state = dmStateBlockSync
		ctx.ExecCtx.LogDebug("change into dmStateBlockSync,ready")
	}
}

func (dm *DecisionMaker) receivedConsensusMaxPeerHeight(ctx *actor.ActionContext) error {
	dm.maxPeerHeights = append(dm.maxPeerHeights, ctx.Messages[0].Data.(uint64))
	return nil
}

func (dm *DecisionMaker) receivedConsensusUp(ctx *actor.ActionContext) error {
	dm.consensusUp = true
	dm.changeStateFromInit(ctx)
	return nil
}

func (dm *DecisionMaker) receivedExtBlockStart(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	switch dm.state {
	case dmStateFastSync:
		if msg.Height >= dm.fastSyncUntil {
			dm.fastSyncCountdown(ctx)
		}

		ctx.ExecCtx.LogDebug("[DecisionMaker.OnMessageArrived] in dmStateFastSync, on MsgExtBlockStart", logger.F("height", msg.Data.(*actor.BlockStart).Height))
		ctx.ExecCtx.Send(scommon.MsgExtAppHash, evmCommon.Hash{}.Bytes())

		dm.changeStateFromFastSync(ctx)
	case dmStateBlockSync:
		blockStart := msg.Data.(*actor.BlockStart)
		blockStart.Coinbase = dm.genesisData.Coinbase //fix coinbase
		blockStart.Extra = dm.genesisData.Extra
		ctx.ExecCtx.Send(scommon.MsgBlockStart, blockStart)
	}
	return nil
}

func (dm *DecisionMaker) receivedExtReapCommand(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	switch dm.state {
	case dmStateFastSync:
		if msg.Height >= dm.fastSyncUntil {
			dm.fastSyncCountdown(ctx)
		}

		dm.changeStateFromFastSync(ctx)
	case dmStateBlockSync:
		ctx.ExecCtx.Send(scommon.MsgReapCommand, msg.Data)
	}
	return nil
}

func (dm *DecisionMaker) receivedExtReapingList(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	switch dm.state {
	case dmStateFastSync:
		if msg.Height >= dm.fastSyncUntil {
			dm.fastSyncCountdown(ctx)
		}

		ctx.ExecCtx.LogDebug("[DecisionMaker.OnMessageArrived] in dmStateFastSync, on MsgExtReapingList")
		ctx.ExecCtx.Send(scommon.MsgMetaBlock, &mtypes.MetaBlock{
			Txs:      [][]byte{},
			Hashlist: []evmCommon.Hash{},
		})

		dm.changeStateFromFastSync(ctx)
	case dmStateBlockSync:
		ctx.ExecCtx.Send(scommon.MsgReapinglist, msg.Data)
	}
	return nil
}

func (dm *DecisionMaker) fastSyncCountdown(ctx *actor.ActionContext) {
	msg := ctx.Messages[0]
	ctx.ExecCtx.LogDebug("[DecisionMaker.OnMessageArrived] fast sync done countdown", logger.F("name", msg.Name), logger.F("height", msg.Height))
	delete(dm.fastSyncMessageTypes, msg.Name)
	dm.updateFSM = true
}
func (dm *DecisionMaker) changeStateFromFastSync(ctx *actor.ActionContext) {
	// FIXME
	if len(dm.fastSyncMessageTypes) == 3 {
		ctx.ExecCtx.LogDebug("[DecisionMaker.OnMessageArrived] switch to dmStateBlockSyncWaiting")
		dm.state = dmStateBlockSyncWaiting
	}
}
func (dm *DecisionMaker) receivedExtBlockEnd(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	switch dm.state {
	case dmStateFastSync:
		if msg.Height+1 >= dm.fastSyncUntil {
			dm.fastSyncCountdown(ctx)
		}

		dm.changeStateFromFastSync(ctx)
	case dmStateBlockSync:
		ctx.ExecCtx.Send(scommon.MsgBlockEnd, msg.Data)
	}
	return nil
}

func (dm *DecisionMaker) receivedExtBlockCompleted(ctx *actor.ActionContext) error {
	msg := ctx.Messages[0]
	switch dm.state {
	case dmStateFastSync:
		if msg.Height+1 >= dm.fastSyncUntil {
			dm.fastSyncCountdown(ctx)
		}

		dm.changeStateFromFastSync(ctx)
	case dmStateBlockSync:
		ctx.ExecCtx.Send(scommon.MsgBlockCompleted, msg.Data)
	}
	return nil
}

func (dm *DecisionMaker) receivedStateSyncDone(ctx *actor.ActionContext) error {
	ctx.ExecCtx.LogDebug("[SyncClient.OnMessageArrived] on MsgStateSyncDone, switch to dmStateBlockSync")
	dm.state = dmStateBlockSync
	return nil
}
func (dm *DecisionMaker) receivedExtTxBlocks(ctx *actor.ActionContext) error {
	ctx.ExecCtx.Send(scommon.MsgTxBlocks, ctx.Messages[0].Data)
	return nil
}
func (dm *DecisionMaker) receivedAppHash(ctx *actor.ActionContext) error {
	ctx.ExecCtx.LogDebug("**************start send MsgExtAppHash")
	ctx.ExecCtx.Send(scommon.MsgExtAppHash, ctx.Messages[0].Data)
	return nil
}

func (dm *DecisionMaker) GetFSMRules() map[int]actor.FSMRule {
	var fastSyncMsgs []string
	for typ := range dm.fastSyncMessageTypes {
		fastSyncMsgs = append(fastSyncMsgs, typ)
	}

	return map[int]actor.FSMRule{
		dmStateUninit: {Accept: []string{
			scommon.MsgInitialization,
			scommon.MsgConsensusMaxPeerHeight,
			scommon.MsgConsensusUp,
		}},
		dmStateFastSync: {Accept: fastSyncMsgs},
		dmStateBlockSyncWaiting: {Accept: []string{
			scommon.MsgConsensusMaxPeerHeight,
			scommon.MsgConsensusUp,
			scommon.MsgStateSyncDone,
		}},
		dmStateBlockSync: {Accept: []string{
			scommon.MsgConsensusMaxPeerHeight,
			scommon.MsgConsensusUp,
			scommon.MsgExtBlockStart,
			scommon.MsgExtBlockEnd,
			scommon.MsgExtReapCommand,
			scommon.MsgExtTxBlocks,
			scommon.MsgExtBlockCompleted,
			scommon.MsgExtReapingList,
			scommon.MsgAppHash,
		}},
	}
}

func (dm *DecisionMaker) GetCurrentState() int {
	if dm.updateFSM {
		dm.updateFSM = false
		return -1
	} else {
		return dm.state
	}
}

func (dm *DecisionMaker) readyToSync() bool {
	return dm.storageUp && (len(dm.maxPeerHeights) >= 10 || dm.consensusUp)
}

func (dm *DecisionMaker) startWithFastSync() (bool, uint64) {
	if dm.consensusUp {
		return false, 0
	}

	maxHeight := uint64(0)
	for _, height := range dm.maxPeerHeights {
		if height > maxHeight {
			maxHeight = height
		}
	}
	return true, maxHeight - 1
}
