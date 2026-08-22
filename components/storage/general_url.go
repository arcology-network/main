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
	"github.com/arcology-network/common-lib/storage/transactional"

	eushared "github.com/arcology-network/eu/shared"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
)

type GeneralUrl struct {
	BasicDBOperation

	generateApcHandle string
	generateUrlUpdate bool
	inited            bool

	objectCached  bool
	apcHandleName string

	outDBMsg                  string
	outGenerationCompletedMsg string
	outPrecommitMsg           string
	outCommitMsg              string

	generateAcctRoot bool

	keys          []string
	encodedValues [][]byte

	metaKeys     []string
	encodedMetas [][]byte
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

func (url *GeneralUrl) PreCommit(ctx *actor.ActionContext, euResults []*eushared.EuResult, height uint64) {
	url.BasicDBOperation.PreCommit(euResults, height)

	if url.generateApcHandle == "generation" {
		ctx.ExecCtx.Send(url.apcHandleName, url.StateStore)
	}
	/*
		if url.generateUrlUpdate {
			keys, values := url.BasicDBOperation.Keys, url.BasicDBOperation.Values
			keys = codec.Strings(keys).Clone()
			encodedValues := make([][]byte, len(values))
			metaKeys := make([]string, len(keys))
			encodedMetas := make([][]byte, len(keys))
			worker := func(start, end, index int, args ...interface{}) {
				for i := start; i < end; i++ {
					if values[i] != nil {
						univalue := values[i].(*statecell.StateCell)
						if univalue.Value() != nil && univalue.Value().(crdtcommon.CRDT).TypeID() == commutative.PATH { // Skip meta data
							metaKeys[i] = keys[i]
							encodedMetas[i] = univalue.Value().(crdtcommon.CRDT).StorageEncode(keys[i])

							keys[i] = ""
							continue
						}
						encodedValues[i] = univalue.Value().(crdtcommon.CRDT).StorageEncode(keys[i])
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

			url.keys = keys
			url.encodedValues = encodedValues
			url.metaKeys = metaKeys
			url.encodedMetas = encodedMetas

			// var na int
			if len(keys) > 0 {
				ctx.ExecCtx.InvokeRPC("transactionalstore", "AddData", &transactional.AddDataRequest{
					Data: &UrlUpdate{
						Keys:          keys,
						EncodedValues: encodedValues,
					},
					RecoverFunc: "urlupdate",
				}, "AddMetas")
			} else {
				url.AddMetas(ctx)
			}
		}
	*/
}

func (url *GeneralUrl) AddMetas(ctx *actor.ActionContext) {
	if len(url.metaKeys) > 0 {
		ctx.ExecCtx.InvokeRPC("transactionalstore", "AddData", &transactional.AddDataRequest{
			Data: &UrlUpdate{
				Keys:          url.metaKeys,
				EncodedValues: url.encodedMetas,
			},
			RecoverFunc: "urlupdate",
		}, "sendAsyncUrlUpdate")
	} else {
		url.sendAsyncUrlUpdate(ctx)
	}
}

func (url *GeneralUrl) sendAsyncUrlUpdate(ctx *actor.ActionContext) {
	ctx.ExecCtx.Send(scommon.MsgUrlUpdate, &UrlUpdate{
		Keys:          url.keys,
		EncodedValues: url.encodedValues,
	})
}

func (url *GeneralUrl) PreCommitCompleted(ctx *actor.ExecutionContext, height uint64) [32]byte {
	if url.generateAcctRoot {
		root := url.BasicDBOperation.PreCommitCompleted(ctx)
		ctx.Send(scommon.MsgAcctHash, root, height)
	}

	return [32]byte{}
}

func (url *GeneralUrl) InitAsync(ctx *actor.ActionContext) {
}

func (url *GeneralUrl) Commit(ctx *actor.ActionContext, height uint64) {
	url.BasicDBOperation.Commit(height)
	if url.objectCached {
		ctx.ExecCtx.Send(scommon.MsgObjectCached, "")
	}

	if url.generateApcHandle == "block" {
		ctx.ExecCtx.Send(url.apcHandleName, url.StateStore)
	}
}

func (url *GeneralUrl) PreCommitAsync(ctx *actor.ExecutionContext) {
	url.BasicDBOperation.PreCommitAsync(ctx)
}

func (url *GeneralUrl) CommitAsync(ctx *actor.ExecutionContext, height uint64) {
	url.BasicDBOperation.CommitAsync(ctx, height)
}

func (url *GeneralUrl) Outputs() map[string]int {
	outputs := make(map[string]int)
	if url.generateApcHandle != "" {
		outputs[url.apcHandleName] = 1
	}
	if url.generateUrlUpdate {
		outputs[scommon.MsgUrlUpdate] = 1
	}
	if url.generateAcctRoot {
		outputs[scommon.MsgAcctHash] = 1
	}
	if url.objectCached {
		outputs[scommon.MsgObjectCached] = 1
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

	if v, ok := params["generate_acct_root"]; !ok {
		panic("parameter not found: generate_acct_root")
	} else {
		url.generateAcctRoot = v.(bool)
	}

	if v, ok := params["object_cached"]; !ok {
		panic("parameter not found: object_cached")
	} else {
		url.objectCached = v.(bool)
	}

}
