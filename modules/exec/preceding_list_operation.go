package exec

import (
	ethcmn "github.com/HPISTechnologies/3rd-party/eth/common"
	cmntyp "github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/component-lib/actor"
)

type PrecedingListOperation struct{}

func (op *PrecedingListOperation) GetData(msg *actor.Message) (hashes []ethcmn.Hash, data []interface{}) {
	results := msg.Data.(*cmntyp.Euresults)
	if results == nil || len(*results) == 0 {
		return
	}

	for _, result := range *results {
		hashes = append(hashes, ethcmn.BytesToHash([]byte(result.H)))
		data = append(data, result)
	}
	return
}

func (op *PrecedingListOperation) GetList(msg *actor.Message) (hashes []ethcmn.Hash) {
	precedings := msg.Data.(*[]*ethcmn.Hash)
	for _, hash := range *precedings {
		hashes = append(hashes, *hash)
	}
	return
}

func (op *PrecedingListOperation) OnListFulfilled(data []interface{}, broker *actor.MessageWrapper) {
	broker.Send(actor.MsgPrecedingsEuresult, data)
}

func (op *PrecedingListOperation) Outputs() map[string]int {
	return map[string]int{actor.MsgPrecedingsEuresult: 1}
}

func (op *PrecedingListOperation) Config(params map[string]interface{}) {}
