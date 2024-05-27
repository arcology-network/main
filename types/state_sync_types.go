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
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

const (
	SlicePerSyncPoint = 256
)

type SyncStatus struct {
	Id        string
	SyncPoint uint64
	Height    uint64

	// FIXME
	RequestId string
}

type SyncPoint struct {
	From       uint64
	To         uint64
	Slices     []ethCommon.Hash
	Parent     *ParentInfo
	SchdStates interface{}
}

type SyncDataRequest struct {
	From  uint64
	To    uint64
	Slice int

	// FIXME
	StartAt time.Time
}

type SyncDataResponse struct {
	SyncDataRequest
	// TBD
	Hash       []byte
	Data       []byte
	Parent     *ParentInfo
	SchdStates interface{}
}

func (sdr *SyncDataRequest) ID() string {
	return fmt.Sprintf("%d|%d", sdr.To, sdr.Slice)
}
