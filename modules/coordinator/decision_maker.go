package coordinator

import (
	"fmt"

	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
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
		actor.MsgStorageUp,
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
		case actor.MsgStorageUp:
			dm.storageUp = true
			dm.initHeight = msg.Height
		case actor.MsgConsensusMaxPeerHeight:
			dm.maxPeerHeights = append(dm.maxPeerHeights, msg.Data.(uint64))
		case actor.MsgConsensusUp:
			dm.consensusUp = true
		}
		if dm.readyToSync() {
			if ok, until := dm.startWithFastSync(); ok {
				fmt.Printf("[DecisionMaker.OnMessageArrived] switch to dmStateFastSync, fast sync until %d\n", until)
				dm.fastSyncUntil = until
				dm.MsgBroker.Send(actor.MsgReapCommand, nil)
				dm.MsgBroker.Send(actor.MsgStateSyncStart, until)
				dm.state = dmStateFastSync
			} else {
				fmt.Printf("[DecisionMaker.OnMessageArrived] switch to dmStateBlockSync\n")
				if dm.initHeight == 0 {
					dm.MsgBroker.Send(actor.MsgBlockEnd, "")
					dm.MsgBroker.Send(actor.MsgBlockCompleted, actor.MsgBlockCompleted_Success)
				}
				dm.MsgBroker.Send(actor.MsgReapCommand, nil)
				dm.state = dmStateBlockSync
			}
		}
	case dmStateFastSync:
		switch msg.Name {
		case actor.MsgExtBlockStart, actor.MsgExtBlockEnd, actor.MsgExtReapCommand, actor.MsgExtReapingList:
			if msg.Height >= dm.fastSyncUntil {
				fmt.Printf("[DecisionMaker.OnMessageArrived] fast sync done countdown, msg name = %s, height = %d\n", msg.Name, msg.Height)
				delete(dm.fastSyncMessageTypes, msg.Name)
				dm.updateFSM = true
			}
		case actor.MsgExtBlockCompleted:
			if msg.Height+1 >= dm.fastSyncUntil {
				fmt.Printf("[DecisionMaker.OnMessageArrived] fast sync done countdown, msg name = %s, height = %d\n", msg.Name, msg.Height)
				delete(dm.fastSyncMessageTypes, msg.Name)
				dm.updateFSM = true
			}
		}

		switch msg.Name {
		case actor.MsgExtBlockStart:
			fmt.Printf("[DecisionMaker.OnMessageArrived] in dmStateFastSync, on MsgExtBlockStart, height = %v\n", msg.Data.(*actor.BlockStart).Height)
			dm.MsgBroker.Send(actor.MsgExtAppHash, ethcmn.Hash{}.Bytes())
		case actor.MsgExtReapingList:
			fmt.Printf("[DecisionMaker.OnMessageArrived] in dmStateFastSync, on MsgExtReapingList\n")
			dm.MsgBroker.Send(actor.MsgMetaBlock, &cmntyp.MetaBlock{
				Txs:      [][]byte{},
				Hashlist: []*ethcmn.Hash{},
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
			dm.MsgBroker.Send(actor.MsgBlockStart, msg.Data)
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
			actor.MsgStorageUp,
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
