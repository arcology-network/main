package storage

import (
	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/storage/transactional"

	"github.com/arcology-network/common-lib/exp/slice"
	eushared "github.com/arcology-network/eu/shared"
	"github.com/arcology-network/storage-committer/commutative"
	"github.com/arcology-network/storage-committer/interfaces"
	univaluepk "github.com/arcology-network/storage-committer/univalue"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type GeneralUrl struct {
	BasicDBOperation

	generateApcHandle string
	generateUrlUpdate bool
	generateAcctRoot  bool
	inited            bool
	cached            bool
	transactional     bool
	apcHandleName     string
}

type UrlUpdate struct {
	Keys          []string
	EncodedValues [][]byte
}

func NewGeneralUrl(apcHandleName string) *GeneralUrl {
	return &GeneralUrl{
		apcHandleName: apcHandleName,
	}
}

func (url *GeneralUrl) PreCommit(euResults []*eushared.EuResult, height uint64) {
	url.BasicDBOperation.PreCommit(euResults, height)

	if url.generateUrlUpdate {
		keys, values := url.BasicDBOperation.Keys, url.BasicDBOperation.Values
		keys = codec.Strings(keys).Clone()
		encodedValues := make([][]byte, len(values))
		metaKeys := make([]string, len(keys))
		encodedMetas := make([][]byte, len(keys))
		worker := func(start, end, index int, args ...interface{}) {
			for i := start; i < end; i++ {
				if values[i] != nil {
					univalue := values[i].(*univaluepk.Univalue)
					if univalue.Value() != nil && univalue.Preexist() && univalue.Value().(interfaces.Type).TypeID() == commutative.PATH { // Skip meta data
						metaKeys[i] = keys[i]
						encodedMetas[i] = univalue.Value().(interfaces.Type).StorageEncode(keys[i])

						keys[i] = ""
						continue
					}
					encodedValues[i] = univalue.Value().(interfaces.Type).StorageEncode(keys[i])
				} else {
					encodedValues[i] = nil
				}
			}
		}
		common.ParallelWorker(len(keys), 4, worker)

		filter := func(_ int, v []byte) bool { return v == nil }
		slice.Remove(&keys, "")
		slice.RemoveIf(&encodedValues, filter)
		slice.Remove(&metaKeys, "")
		slice.RemoveIf(&encodedMetas, filter)

		var na int
		if len(keys) > 0 {
			intf.Router.Call("transactionalstore", "AddData", &transactional.AddDataRequest{
				Data: &UrlUpdate{
					Keys:          keys,
					EncodedValues: encodedValues,
				},
				RecoverFunc: "urlupdate",
			}, &na)
		}
		if len(metaKeys) > 0 {
			intf.Router.Call("transactionalstore", "AddData", &transactional.AddDataRequest{
				Data: &UrlUpdate{
					Keys:          metaKeys,
					EncodedValues: encodedMetas,
				},
				RecoverFunc: "urlupdate",
			}, &na)
		}

		url.MsgBroker.Send(actor.MsgUrlUpdate, &UrlUpdate{
			Keys:          keys,
			EncodedValues: encodedValues,
		})
	}
	if url.transactional {
		url.MsgBroker.Send(actor.MsgTransactionalAddCompleted, "ok")
	}

	if url.generateApcHandle == "generation" {
		url.MsgBroker.Send(url.apcHandleName, url.StateStore)
	}
}

func (url *GeneralUrl) Commit(height uint64) {
	if url.generateAcctRoot {
		url.MsgBroker.Send(actor.MsgAcctHash, url.BasicDBOperation.stateRoot)
	}

	if !url.inited {
		url.inited = true
		if url.generateApcHandle == "generation" {
			url.MsgBroker.Send(url.apcHandleName, url.StateStore)
		}
	} else {
		url.BasicDBOperation.Commit(height)
	}

	if url.generateApcHandle == "block" {
		url.MsgBroker.Send(url.apcHandleName, url.StateStore)
	}

}

func (url *GeneralUrl) Outputs() map[string]int {
	outputs := make(map[string]int)
	if url.generateApcHandle != "" {
		outputs[url.apcHandleName] = 1
	}
	if url.generateUrlUpdate {
		outputs[actor.MsgUrlUpdate] = 1
	}
	if url.generateAcctRoot {
		outputs[actor.MsgAcctHash] = 1
	}
	if url.cached {
		outputs[actor.MsgCached] = 1
	}
	if url.transactional {
		outputs[actor.MsgTransactionalAddCompleted] = 1
	}

	return outputs
}

func (url *GeneralUrl) Config(params map[string]interface{}) {
	if v, ok := params["generate_apc_handle"]; !ok {
		panic("parameter not found: generate_apc_handle")
	} else {
		url.generateApcHandle = v.(string)
	}

	if v, ok := params["generate_url_update"]; !ok {
		panic("parameter not found: generate_url_update")
	} else {
		url.generateUrlUpdate = v.(bool)
	}

	if v, ok := params["generate_acct_root"]; !ok {
		panic("parameter not found: generate_acct_root")
	} else {
		url.generateAcctRoot = v.(bool)
	}

	if v, ok := params["cached"]; !ok {
		panic("parameter not found: cached")
	} else {
		url.cached = v.(bool)
	}

	if v, ok := params["transactional"]; !ok {
		panic("parameter not found: transactional")
	} else {
		url.transactional = v.(bool)
	}
}
