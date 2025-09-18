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
	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/storage/transactional"

	"github.com/arcology-network/common-lib/exp/slice"
	eushared "github.com/arcology-network/eu/shared"
	interfaces "github.com/arcology-network/storage-committer/common"
	"github.com/arcology-network/storage-committer/type/commutative"
	univaluepk "github.com/arcology-network/storage-committer/type/univalue"
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type GeneralUrl struct {
	BasicDBOperation

	generateApcHandle string
	generateUrlUpdate bool
	// generateAcctRoot  bool
	inited        bool
	cached        bool
	transactional bool
	objectCached  bool
	apcHandleName string

	outDBMsg                  string
	outGenerationCompletedMsg string
	outPrecommitMsg           string
	outCommitMsg              string
}

type UrlUpdate struct {
	Keys          []string
	EncodedValues [][]byte
}

func NewGeneralUrl(apcHandleName, outDBMsg, outGenerationCompletedMsg, outPrecommitMsg, outCommitMsg string) *GeneralUrl {
	return &GeneralUrl{
		apcHandleName:             apcHandleName,
		outDBMsg:                  outDBMsg,
		outGenerationCompletedMsg: outGenerationCompletedMsg,
		outPrecommitMsg:           outPrecommitMsg,
		outCommitMsg:              outCommitMsg,
	}
}

func (url *GeneralUrl) PreCommit(euResults []*eushared.EuResult, height uint64) {
	if url.generateApcHandle == "generation" {
		RequestLock("exec", "GeneralUrl")
		defer ReleaseLock("exec", "GeneralUrl")
	}

	url.BasicDBOperation.PreCommit(euResults, height)
	url.MsgBroker.Send(url.outPrecommitMsg, "")

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
					if univalue.Value() != nil && univalue.Value().(interfaces.Type).TypeID() == commutative.PATH { // Skip meta data
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

	// if url.transactional {
	// 	fmt.Printf("======components/storage/general_url.go  PreCommit url.transactional:%v\n", url.transactional)
	// 	url.MsgBroker.Send(actor.MsgTransactionalAddCompleted, "ok")
	// }
	if url.generateApcHandle == "generation" {
		url.MsgBroker.Send(url.apcHandleName, url.StateStore)
	}
}

func (url *GeneralUrl) PreCommitCompleted() {
	url.BasicDBOperation.PreCommitCompleted()
	url.MsgBroker.Send(url.outGenerationCompletedMsg, "")
}

func (url *GeneralUrl) InitAsync() {
	url.MsgBroker.Send(url.outDBMsg, url.BasicDBOperation.StateStore)
}

func (url *GeneralUrl) Commit(height uint64) {
	url.BasicDBOperation.Commit(height)
	url.MsgBroker.Send(url.outCommitMsg, "")
	if url.objectCached {
		url.MsgBroker.Send(actor.MsgObjectCached, "")
	}

	if url.generateApcHandle == "block" {
		url.MsgBroker.Send(url.apcHandleName, url.StateStore)
	}

}

func (url *GeneralUrl) Outputs() map[string]int {
	outputs := make(map[string]int)
	if url.generateApcHandle != "" {
		outputs[url.apcHandleName] = 1
		outputs[actor.MsgApcHandleInit] = 1
	}
	if url.generateUrlUpdate {
		outputs[actor.MsgUrlUpdate] = 1
	}
	// if url.generateAcctRoot {
	// 	outputs[actor.MsgAcctHash] = 1
	// }
	if url.cached {
		outputs[actor.MsgCached] = 1
	}
	if url.objectCached {
		outputs[actor.MsgObjectCached] = 1
	}
	if url.transactional {
		outputs[actor.MsgTransactionalAddCompleted] = 1
	}

	outputs[url.outDBMsg] = 1
	outputs[url.outGenerationCompletedMsg] = 1
	outputs[url.outPrecommitMsg] = 1
	outputs[url.outCommitMsg] = 1
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

	// if v, ok := params["generate_acct_root"]; !ok {
	// 	panic("parameter not found: generate_acct_root")
	// } else {
	// 	url.generateAcctRoot = v.(bool)
	// }

	if v, ok := params["cached"]; !ok {
		panic("parameter not found: cached")
	} else {
		url.cached = v.(bool)
	}
	if v, ok := params["object_cached"]; !ok {
		panic("parameter not found: object_cached")
	} else {
		url.objectCached = v.(bool)
	}

	if v, ok := params["transactional"]; !ok {
		panic("parameter not found: transactional")
	} else {
		url.transactional = v.(bool)
	}
}
