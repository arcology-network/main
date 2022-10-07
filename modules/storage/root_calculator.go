package storage

import (
	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
	"github.com/HPISTechnologies/component-lib/storage"
	urlcmn "github.com/HPISTechnologies/concurrenturl/v2/common"
	urltyp "github.com/HPISTechnologies/concurrenturl/v2/type"
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
		actor.MsgBlockCompleted,
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
	switch msg.Name {
	case actor.MsgParentInfo:
		rc.lastRoot = msg.Data.(*cmntyp.ParentInfo).ParentRoot
		// TODO
		rc.state = rcStateRunning
	case actor.MsgEuResults:
		data := msg.Data.(*cmntyp.Euresults)
		_, transitions := storage.GetTransitions(*data)
		rc.merkle.Import(transitions)
	case actor.CombinedName(actor.MsgListFulfilled, actor.MsgUrlUpdate):
		combined := msg.Data.(*actor.CombinerElements)
		urlUpdate := combined.Get(actor.MsgUrlUpdate).Data.(*storage.UrlUpdate)
		rc.lastRoot = calcRootHash(rc.merkle, rc.lastRoot, urlUpdate.Keys, urlUpdate.EncodedValues)
		rc.MsgBroker.Send(actor.MsgAcctHash, &rc.lastRoot)
	case actor.MsgBlockCompleted:
		rc.merkle.Clear()
	}
	return nil
}

func (rc *RootCalculator) GetStateDefinitions() map[int][]string {
	return map[int][]string{
		rcStateUninit: {actor.MsgParentInfo},
		rcStateRunning: {
			actor.MsgEuResults,
			actor.CombinedName(actor.MsgListFulfilled, actor.MsgUrlUpdate),
			actor.MsgBlockCompleted,
		},
	}
}

func (rc *RootCalculator) GetCurrentState() int {
	return rc.state
}
