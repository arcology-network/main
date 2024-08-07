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
	"fmt"

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	evmCommon "github.com/ethereum/go-ethereum/common"
)

const (
	dmStateUninit = iota
	dmStateFastSync
	dmStateBlockSyncWaiting
	dmStateBlockSync
)

type DecisionMaker struct {
	actor.WorkerThread

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

func NewDecisionMaker(concurrency int, groupId string) actor.IWorkerEx {
	dm := &DecisionMaker{
		state: dmStateUninit,
		fastSyncMessageTypes: map[string]struct{}{
			actor.MsgConsensusMaxPeerHeight: {},
			actor.MsgConsensusUp:            {},
			actor.MsgExtBlockStart:          {},
			actor.MsgExtBlockEnd:            {},
			actor.MsgExtReapCommand:         {},
			actor.MsgExtTxBlocks:            {},
			actor.MsgExtReapingList:         {},
			actor.MsgExtBlockCompleted:      {},
		},
	}
	dm.Set(concurrency, groupId)
	return dm
}

func (dm *DecisionMaker) Inputs() ([]string, bool) {
	return []string{
		actor.MsgInitialization,
		actor.MsgConsensusMaxPeerHeight,
		actor.MsgConsensusUp,
		actor.MsgExtBlockStart,
		actor.MsgExtBlockEnd,
		actor.MsgExtReapCommand,
		actor.MsgExtTxBlocks,
		actor.MsgExtBlockCompleted,
		actor.MsgExtReapingList,
		actor.MsgAppHash,
		actor.MsgStateSyncDone,
	}, false
}

func (dm *DecisionMaker) Outputs() map[string]int {
	return map[string]int{
		actor.MsgBlockStart:     1,
		actor.MsgBlockEnd:       1,
		actor.MsgReapCommand:    1,
		actor.MsgTxBlocks:       1,
		actor.MsgBlockCompleted: 1,
		actor.MsgReapinglist:    1,
		actor.MsgMetaBlock:      1,
		actor.MsgExtAppHash:     1,
		actor.MsgStateSyncStart: 1,
	}
}

func (dm *DecisionMaker) OnStart() {}

func (dm *DecisionMaker) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	switch dm.state {
	case dmStateUninit:
		switch msg.Name {
		case actor.MsgInitialization:
			dm.storageUp = true
			dm.initHeight = msg.Height
			initialization := msg.Data.(*mtypes.Initialization)
			dm.genesisData = initialization.BlockStart
		case actor.MsgConsensusMaxPeerHeight:
			dm.maxPeerHeights = append(dm.maxPeerHeights, msg.Data.(uint64))
		case actor.MsgConsensusUp:
			dm.consensusUp = true
		}
		// if dm.readyToSync() {
		// 	if ok, until := dm.startWithFastSync(); ok {
		// 		fmt.Printf("[DecisionMaker.OnMessageArrived] switch to dmStateFastSync, fast sync until %d\n", until)
		// 		dm.fastSyncUntil = until
		// 		dm.MsgBroker.Send(actor.MsgReapCommand, nil)
		// 		dm.MsgBroker.Send(actor.MsgStateSyncStart, until)
		// 		dm.state = dmStateFastSync
		// 	} else {
		// 		fmt.Printf("[DecisionMaker.OnMessageArrived] switch to dmStateBlockSync\n")
		// 		if dm.initHeight == 0 {
		// 			dm.MsgBroker.Send(actor.MsgBlockEnd, "")
		// 			dm.MsgBroker.Send(actor.MsgBlockCompleted, actor.MsgBlockCompleted_Success)
		// 		}
		// 		dm.MsgBroker.Send(actor.MsgReapCommand, nil)
		// 		dm.state = dmStateBlockSync
		// 	}
		// }

		if dm.storageUp && dm.consensusUp {
			dm.MsgBroker.Send(actor.MsgReapCommand, nil)
			dm.state = dmStateBlockSync
			dm.AddLog(log.LogLevel_Debug, ">>>>>change into dmStateBlockSync,ready ************************")
		}
	case dmStateFastSync:
		switch msg.Name {
		case actor.MsgExtBlockStart, actor.MsgExtReapCommand, actor.MsgExtReapingList:
			if msg.Height >= dm.fastSyncUntil {
				fmt.Printf("[DecisionMaker.OnMessageArrived] fast sync done countdown, msg name = %s, height = %d\n", msg.Name, msg.Height)
				delete(dm.fastSyncMessageTypes, msg.Name)
				dm.updateFSM = true
			}
		case actor.MsgExtBlockEnd, actor.MsgExtBlockCompleted:
			if msg.Height+1 >= dm.fastSyncUntil {
				fmt.Printf("[DecisionMaker.OnMessageArrived] fast sync done countdown, msg name = %s, height = %d\n", msg.Name, msg.Height)
				delete(dm.fastSyncMessageTypes, msg.Name)
				dm.updateFSM = true
			}
		}

		switch msg.Name {
		case actor.MsgExtBlockStart:
			fmt.Printf("[DecisionMaker.OnMessageArrived] in dmStateFastSync, on MsgExtBlockStart, height = %v\n", msg.Data.(*actor.BlockStart).Height)
			dm.MsgBroker.Send(actor.MsgExtAppHash, evmCommon.Hash{}.Bytes())
		case actor.MsgExtReapingList:
			fmt.Printf("[DecisionMaker.OnMessageArrived] in dmStateFastSync, on MsgExtReapingList\n")
			dm.MsgBroker.Send(actor.MsgMetaBlock, &mtypes.MetaBlock{
				Txs:      [][]byte{},
				Hashlist: []evmCommon.Hash{},
			})
		}

		// FIXME
		if len(dm.fastSyncMessageTypes) == 3 {
			fmt.Printf("[DecisionMaker.OnMessageArrived] switch to dmStateBlockSyncWaiting\n")
			dm.state = dmStateBlockSyncWaiting
		}
	case dmStateBlockSyncWaiting:
		switch msg.Name {
		case actor.MsgStateSyncDone:
			fmt.Printf("[SyncClient.OnMessageArrived] on MsgStateSyncDone, switch to dmStateBlockSync\n")
			dm.state = dmStateBlockSync
		}
	case dmStateBlockSync:
		switch msg.Name {
		case actor.MsgExtBlockStart:
			blockStart := msg.Data.(*actor.BlockStart)
			blockStart.Coinbase = dm.genesisData.Coinbase //fix coinbase
			blockStart.Extra = dm.genesisData.Extra
			dm.MsgBroker.Send(actor.MsgBlockStart, blockStart)
		case actor.MsgExtBlockEnd:
			dm.MsgBroker.Send(actor.MsgBlockEnd, msg.Data)
		case actor.MsgExtReapCommand:
			dm.MsgBroker.Send(actor.MsgReapCommand, msg.Data)
		case actor.MsgExtTxBlocks:
			dm.MsgBroker.Send(actor.MsgTxBlocks, msg.Data)
		case actor.MsgExtBlockCompleted:
			dm.MsgBroker.Send(actor.MsgBlockCompleted, msg.Data)
		case actor.MsgExtReapingList:
			dm.MsgBroker.Send(actor.MsgReapinglist, msg.Data)
		case actor.MsgAppHash:
			dm.MsgBroker.Send(actor.MsgExtAppHash, msg.Data)
		}
	}

	return nil
}

func (dm *DecisionMaker) GetStateDefinitions() map[int][]string {
	var fastSyncMsgs []string
	for typ := range dm.fastSyncMessageTypes {
		fastSyncMsgs = append(fastSyncMsgs, typ)
	}
	return map[int][]string{
		dmStateUninit: {
			actor.MsgInitialization,
			actor.MsgConsensusMaxPeerHeight,
			actor.MsgConsensusUp,
		},
		dmStateFastSync: fastSyncMsgs,
		dmStateBlockSyncWaiting: {
			actor.MsgConsensusMaxPeerHeight,
			actor.MsgConsensusUp,
			actor.MsgStateSyncDone,
		},
		dmStateBlockSync: {
			actor.MsgConsensusMaxPeerHeight,
			actor.MsgConsensusUp,
			actor.MsgExtBlockStart,
			actor.MsgExtBlockEnd,
			actor.MsgExtReapCommand,
			actor.MsgExtTxBlocks,
			actor.MsgExtBlockCompleted,
			actor.MsgExtReapingList,
			actor.MsgAppHash,
		},
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
