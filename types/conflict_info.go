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
	"github.com/ethereum/go-ethereum/common"
)

const (
	signSize         = 4
	ConflictInfoSize = common.AddressLength + signSize + common.AddressLength + signSize
)

type ConflictInfo struct {
	LeftAddress  common.Address
	LeftSign     [signSize]byte
	RightAddress common.Address
	RightSign    [signSize]byte
}

type ConflictInfos struct {
	Conflicts []*ConflictInfo
	Bitmap    map[[ConflictInfoSize]byte]interface{}
}

func NewConflictInfos() *ConflictInfos {
	return &ConflictInfos{
		Conflicts: []*ConflictInfo{},
		Bitmap:    map[[ConflictInfoSize]byte]interface{}{},
	}
}

func (cs *ConflictInfos) Reset() {
	cs.Bitmap = map[[ConflictInfoSize]byte]interface{}{}
	cs.Conflicts = []*ConflictInfo{}
}

func (cs *ConflictInfos) Format() ([]common.Address, []common.Address, [][4]byte, [][4]byte) {
	size := len(cs.Conflicts)
	leftAddrs := make([]common.Address, size)
	rightAddrs := make([]common.Address, size)
	leftSigns := make([][4]byte, size)
	rigthSigns := make([][4]byte, size)
	for i, ci := range cs.Conflicts {
		leftAddrs[i] = ci.LeftAddress
		rightAddrs[i] = ci.RightAddress
		leftSigns[i] = ci.LeftSign
		rigthSigns[i] = ci.RightSign
	}
	return leftAddrs, rightAddrs, leftSigns, rigthSigns
}

func (cs *ConflictInfos) Add(ci *ConflictInfo) {
	buffer := make([]byte, ConflictInfoSize)

	size := copy(buffer[:common.AddressLength], ci.LeftAddress[:])
	size += copy(buffer[size:size+signSize], ci.LeftSign[:])
	size += copy(buffer[size:size+common.AddressLength], ci.RightAddress[:])
	size += copy(buffer[size:size+signSize], ci.RightSign[:])
	if _, ok := cs.Bitmap[[ConflictInfoSize]byte(buffer)]; !ok {
		cs.Bitmap[[ConflictInfoSize]byte(buffer)] = 1
		cs.Conflicts = append(cs.Conflicts, ci)
	}
}
func (cs *ConflictInfos) GobEncode() ([]byte, error) {
	buffer := make([]byte, ConflictInfoSize*len(cs.Conflicts))
	idx := 0
	for k, _ := range cs.Bitmap {
		copy(buffer[idx*ConflictInfoSize:(idx+1)*ConflictInfoSize], k[:])
		idx += 1
	}
	return buffer, nil
}
func (cs *ConflictInfos) GobDecode(data []byte) error {
	bitmap := make(map[[48]byte]interface{}, len(data)/ConflictInfoSize)
	for i := 0; i < len(bitmap); i++ {
		buffer := make([]byte, ConflictInfoSize)
		copy(buffer[:], data[i*ConflictInfoSize:(i+1)*ConflictInfoSize])
		bitmap[[ConflictInfoSize]byte(buffer)] = 1
	}
	cs.Bitmap = bitmap
	return nil
}
