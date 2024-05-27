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
	"encoding/gob"
)

// var arEncoder *arbReqEncoder
// var arDecoder *arbReqDecoder

// var bytesPool chan []byte

func init() {
	gob.Register(&RPCTransaction{})
	gob.Register(&RPCBlock{})

	gob.Register(&ArbitratorRequest{})
	gob.Register(&ArbitratorResponse{})

	gob.Register(&BlockResult{})
	gob.Register(&MetaBlock{})
	gob.Register(&MonacoBlock{})

	gob.Register(&ParentInfo{})
	gob.Register(&SyncStatus{})
	gob.Register(&SyncPoint{})
	gob.Register(&SyncDataRequest{})
	gob.Register(&SyncDataResponse{})

	gob.Register(RequestBalance{})
	gob.Register(RequestContainer{})
	gob.Register(&RequestBlock{})
	gob.Register(&RequestReceipt{})
	gob.Register(Block{})
	gob.Register(Log{})
	gob.Register([]*QueryReceipt{})
	gob.Register(&RequestParameters{})
	gob.Register(&RequestBlockEth{})
	gob.Register(&RequestStorage{})

	gob.Register(&StatisticalInformation{})
	gob.Register(&ExecutorRequest{})
	gob.Register(&ExecutorResponses{})

	// arEncoder = newArbReqEncoder()
	// arDecoder = newArbReqDecoder()

	// bytesPool = make(chan []byte, 100)
	// for i := 0; i < 100; i++ {
	// 	bytesPool <- make([]byte, 0, 2*1024*1024)
	// }

}
