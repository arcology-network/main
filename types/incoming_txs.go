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

package types

import (
	"fmt"
	"strings"
)

const (
	TxSourceLocal = iota
	TxSourceConsensus
	TxSourceMonacoP2p
)

var (
	SourceTypeStr = map[int]string{
		TxSourceLocal:     "loc",
		TxSourceConsensus: "con",
		TxSourceMonacoP2p: "p2p",
	}
)

type TxSource string

func NewTxSource(typ int, id string) TxSource {
	return TxSource(fmt.Sprintf("%s:%s", SourceTypeStr[typ], id))
}

func (src TxSource) BypassRepeatCheck() bool {
	return strings.HasPrefix(string(src), SourceTypeStr[TxSourceMonacoP2p])
}

func (src TxSource) IsForWaitingList() bool {
	return strings.HasPrefix(string(src), SourceTypeStr[TxSourceMonacoP2p])
}

type IncomingTxs struct {
	Txs       [][]byte
	Src       TxSource
	RequestID string
}
