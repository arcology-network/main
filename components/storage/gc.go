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
	scommon "github.com/arcology-network/streamer/common"
	"github.com/arcology-network/streamer/logger"
)

type Gc struct {
}

func NewGc() actor.Business {
	gc := Gc{}
	return &gc
}

func (gc *Gc) Inputs() ([]string, bool) {
	return []string{scommon.MsgGc}, false
}

func (gc *Gc) Outputs() map[string]int {
	return map[string]int{}
}

func (gc *Gc) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgGc, gc.gc)
}

func (gc *Gc) gc(ctx *actor.ActionContext) error {
	t := time.Now()
	runtime.GC()
	ctx.ExecCtx.LogDebug("gc completed", logger.F("time", time.Since(t)))
	return nil
}
