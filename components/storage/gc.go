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
	"runtime"
	"time"

	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	"go.uber.org/zap"
)

type Gc struct {
	actor.WorkerThread
}

func NewGc(lanes int, groupid string) actor.IWorkerEx {
	gc := Gc{}
	gc.Set(lanes, groupid)
	return &gc
}

func (gc *Gc) Inputs() ([]string, bool) {
	return []string{actor.MsgGc}, false
}

func (gc *Gc) Outputs() map[string]int {
	return map[string]int{}
}

func (gc *Gc) OnStart() {
}

func (gc *Gc) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgGc:
			t := time.Now()
			runtime.GC()
			gc.AddLog(log.LogLevel_Info, "gc completed ---->", zap.Duration("time", time.Since(t)))
		}
	}
	return nil
}
