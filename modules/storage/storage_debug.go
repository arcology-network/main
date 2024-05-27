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
	"github.com/arcology-network/streamer/actor"
	intf "github.com/arcology-network/streamer/interface"
)

type StorageDebug struct {
	actor.WorkerThread
}

// return a Subscriber struct
func NewStorageDebug(concurrency int, groupid string) actor.IWorkerEx {
	s := StorageDebug{}
	s.Set(concurrency, groupid)
	return &s
}

func (s *StorageDebug) Inputs() ([]string, bool) {
	return []string{actor.MsgExecutingLogs}, false
}

func (s *StorageDebug) Outputs() map[string]int {
	return map[string]int{}
}

func (*StorageDebug) OnStart() {}
func (*StorageDebug) Stop()    {}

func (s *StorageDebug) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgExecutingLogs:
			executingLogs := v.Data.([]string)
			var na int
			intf.Router.Call("debugstore", "SaveLog", &LogSaveRequest{
				Height:   v.Height,
				Execlogs: executingLogs,
			}, &na)
		}
	}
	return nil
}
