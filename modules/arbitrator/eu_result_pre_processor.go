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

package arbitrator

import (
	"github.com/arcology-network/common-lib/common"
	eushared "github.com/arcology-network/eu/shared"
	"github.com/arcology-network/main/modules/arbitrator/types"
	"github.com/arcology-network/streamer/actor"
)

type EuResultPreProcessor struct {
	actor.WorkerThread
}

func NewEuResultPreProcessor(concurrency int, groupid string) actor.IWorkerEx {
	p := &EuResultPreProcessor{}
	p.Set(concurrency, groupid)
	return p
}

func (p *EuResultPreProcessor) OnStart() {

}

func (p *EuResultPreProcessor) Inputs() ([]string, bool) {
	return []string{actor.MsgTxAccessRecords}, false
}

func (p *EuResultPreProcessor) Outputs() map[string]int {
	return map[string]int{
		actor.MsgPreProcessedImportEuResults: 100,
	}
}

func (p *EuResultPreProcessor) OnMessageArrived(msgs []*actor.Message) error {
	results := *(msgs[0].Data.(*eushared.TxAccessRecordSet))

	processed := make([]*types.AccessRecord, len(results))
	worker := func(start, end, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			processed[i] = types.Decode(results[i])
		}
	}
	common.ParallelWorker(len(results), p.Concurrency, worker)

	p.MsgBroker.Send(actor.MsgPreProcessedImportEuResults, processed)
	return nil
}
