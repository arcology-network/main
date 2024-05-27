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

package ethapi

import (
	"fmt"

	"github.com/arcology-network/streamer/actor"
)

type Coinbase struct {
	actor.WorkerThread
}

// return a Subscriber struct
func NewCoinbase(concurrency int, groupid string) actor.IWorkerEx {
	coinbase := Coinbase{}
	coinbase.Set(concurrency, groupid)
	return &coinbase
}

func (sq *Coinbase) Inputs() ([]string, bool) {
	return []string{
		actor.MsgCoinbase,
	}, false
}

func (sq *Coinbase) Outputs() map[string]int {
	return map[string]int{}
}

func (sq *Coinbase) Config(params map[string]interface{}) {

}

func (*Coinbase) OnStart() {

}

func (*Coinbase) Stop() {}

func (sq *Coinbase) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgCoinbase:
			coinbase := v.Data.(*actor.BlockStart)
			options.Coinbase = fmt.Sprintf("0x%x", coinbase.Coinbase.Bytes())
		}
	}
	return nil
}
