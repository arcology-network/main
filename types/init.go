package types

import (
	"encoding/gob"
)

var arEncoder *arbReqEncoder
var arDecoder *arbReqDecoder

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

	arEncoder = newArbReqEncoder()
	arDecoder = newArbReqDecoder()

	// bytesPool = make(chan []byte, 100)
	// for i := 0; i < 100; i++ {
	// 	bytesPool <- make([]byte, 0, 2*1024*1024)
	// }

}
