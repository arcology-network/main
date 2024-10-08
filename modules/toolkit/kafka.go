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

package toolkit

import (
	"fmt"
	"strings"

	eushared "github.com/arcology-network/eu/shared"
	commutative "github.com/arcology-network/storage-committer/type/commutative"
	"github.com/arcology-network/streamer/actor"
)

type kafka struct {
	actor.WorkerThread
	queryHeight uint64
	queryKey    string
}

// return a Subscriber struct
func Newkafka(concurrency int, groupid string) actor.IWorkerEx {
	ka := kafka{}
	ka.Set(concurrency, groupid)
	return &ka
}
func (c *kafka) Config(params map[string]interface{}) {
	c.queryHeight = uint64(params["queryheight"].(float64))
	c.queryKey = params["querykey"].(string)
}
func (c *kafka) Inputs() ([]string, bool) {
	return []string{actor.MsgEuResults}, false
}

func (c *kafka) Outputs() map[string]int {
	return map[string]int{}
}

func (c *kafka) OnStart() {
}

func (c *kafka) OnMessageArrived(msgs []*actor.Message) error {
	total := 0
	cc := 0
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgEuResults:
			if v.Height == c.queryHeight {
				data := v.Data.(*eushared.Euresults)
				if data != nil {
					for i := range *data {
						transitions := (*data)[i].Trans

						transitionData := transitions //univaluepk.Univalues{}.Decode(transitions).(univaluepk.Univalues)
						size := 0

						for j := range transitionData {
							key := *transitionData[j].GetPath()
							if strings.Contains(key, c.queryKey) {
								size = size + 1
								total = total + 1
								nonce := transitionData[j].Value().(*commutative.Uint64).Delta().(uint64)

								if nonce > 1 {
									fmt.Printf("=====height=%v======h=%x   %v\n", v.Height, []byte((*data)[i].H), nonce)
								} else if nonce == 1 {
									cc = cc + 1
								}

							}
						}
						if size > 1 {
							fmt.Printf("=====height=%v======h=%x   %v\n", v.Height, []byte((*data)[i].H), size)
						}
					}
				}
			}
			fmt.Printf("======height=%v  total:%v cc:%v\n", v.Height, total, cc)
		}
	}

	return nil
}
