package storage

import (
	ethcmn "github.com/arcology-network/3rd-party/eth/common"
	cmntyp "github.com/arcology-network/common-lib/types"
	"github.com/arcology-network/component-lib/actor"
	"github.com/arcology-network/component-lib/storage"
	urlcmn "github.com/arcology-network/concurrenturl/v2/common"
	urltyp "github.com/arcology-network/concurrenturl/v2/type"
)

const (
	rcStateUninit = iota
	rcStateRunning
)

type RootCalculator struct {
	actor.WorkerThread

	merkle   *urltyp.AccountMerkle
	lastRoot ethcmn.Hash
	state    int
}

func NewRootCalculator(concurrency int, groupId string) actor.IWorkerEx {
	rc := &RootCalculator{
		merkle: urltyp.NewAccountMerkle(urlcmn.NewPlatform()),
		state:  rcStateUninit,
	}
	rc.Set(concurrency, groupId)
	return rc
}

func (rc *RootCalculator) Inputs() ([]string, bool) {
	return []string{
		actor.MsgParentInfo,
		actor.MsgEuResults,
		actor.CombinedName(actor.MsgListFulfilled, actor.MsgUrlUpdate),
		actor.MsgBlockEnd,
	}, false
}

func (rc *RootCalculator) Outputs() map[string]int {
	return map[string]int{
		actor.MsgAcctHash: 1,
	}
}

func (rc *RootCalculator) OnStart() {}

func (rc *RootCalculator) OnMessageArrived(msgs []*actor.Message) error {
	msg := msgs[0]
	if rc.state == rcStateUninit {
		rc.lastRoot = msg.Data.(*cmntyp.ParentInfo).ParentRoot
		rc.state = rcStateRunning
	} else {
		switch msg.Name {
		case actor.MsgEuResults:
			data := msg.Data.(*cmntyp.Euresults)
			_, transitions := storage.GetTransitions(*data)
			rc.merkle.Import(transitions)
		case actor.CombinedName(actor.MsgListFulfilled, actor.MsgUrlUpdate):
			rc.CheckPoint("start calculate acchash")
			combined := msg.Data.(*actor.CombinerElements)
			urlUpdate := combined.Get(actor.MsgUrlUpdate).Data.(*storage.UrlUpdate)
			rc.lastRoot = calcRootHash(rc.merkle, rc.lastRoot, urlUpdate.Keys, urlUpdate.EncodedValues)
			rc.MsgBroker.Send(actor.MsgAcctHash, &rc.lastRoot)
			rc.CheckPoint("acchash calculate completed")
		case actor.MsgBlockEnd:
			rc.merkle.Clear()
		}
	}
	return nil
}

func (rc *RootCalculator) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		rcStateUninit: {actor.MsgParentInfo},
		rcStateRunning: {
			actor.MsgEuResults,
			actor.CombinedName(actor.MsgListFulfilled, actor.MsgUrlUpdate),
			actor.MsgBlockEnd,
			actor.MsgParentInfo, // Receive and ignore.
		},
	}
}

func (rc *RootCalculator) GetCurrentState() int {
	return rc.state
}
