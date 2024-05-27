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
	"time"

	codec "github.com/arcology-network/common-lib/codec"
)

type StatisticalInformation struct {
	Key      string
	Value    string
	TimeUsed time.Duration
}

func (si StatisticalInformation) EncodeToBytes() []byte {
	data := [][]byte{
		[]byte(si.Key),
		[]byte(si.Value),
		// common.Uint64ToBytes(common.Int64ToUint64(int64(si.TimeUsed))),
	}
	return codec.Byteset(data).Encode()
}
func (si *StatisticalInformation) Decode(data []byte) error {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	si.Key = string(fields[0])
	si.Value = string(fields[1])
	// si.TimeUsed = time.Duration(common.Uint64ToInt64(common.BytesToUint64(fields[2])))
	return nil
}
