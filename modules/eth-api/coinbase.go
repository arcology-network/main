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

	mtypes "github.com/arcology-network/main/types"
	"github.com/arcology-network/streamer/actor"
	scommon "github.com/arcology-network/streamer/common"
)

type Coinbase struct {
}

// return a Subscriber struct
func NewCoinbase() actor.Business {
	coinbase := Coinbase{}

	return &coinbase
}

func (sq *Coinbase) Inputs() ([]string, bool) {
	return []string{
		scommon.MsgInitialization,
	}, false
}

func (sq *Coinbase) Outputs() map[string]int {
	return map[string]int{}
}

func (sq *Coinbase) RegisterActions(reg actor.ActionRegistrar) {
	reg.Register(scommon.MsgInitialization, sq.SetCoin)
}

func (sq *Coinbase) SetCoin(ctx *actor.ActionContext) error {
	coinbase := ctx.Messages[0].Data.(*mtypes.Initialization).BlockStart
	options.Coinbase = fmt.Sprintf("0x%x", coinbase.Coinbase.Bytes())

	return nil
}
