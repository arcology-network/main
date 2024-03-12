package types

import (
	codec "github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/common-lib/common"
	"github.com/arcology-network/common-lib/exp/slice"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type ArbitratorRequest struct {
	TxsListGroup [][]*TxElement
}

type TxElement struct {
	TxHash  *ethCommon.Hash
	Batchid uint64
	Txid    uint32
}

func (this TxElement) Encode() []byte {
	tmpData := [][]byte{
		this.TxHash[:],
		codec.Uint64(this.Batchid).Encode(),
		codec.Uint32(this.Txid).Encode(),
	}
	return codec.Byteset(tmpData).Encode()
}

func (this *TxElement) Decode(data []byte) *TxElement {
	fields := codec.Byteset{}.Decode(data).(codec.Byteset)
	hash := ethCommon.BytesToHash(fields[0])
	this.TxHash = &hash
	this.Batchid = uint64(new(codec.Uint64).Decode(fields[1]).(codec.Uint64))
	this.Txid = uint32(new(codec.Uint32).Decode(fields[2]).(codec.Uint32))
	return this
}

func (tx TxElement) Size() uint32 {
	return uint32(ethCommon.HashLength) + codec.Uint64(0).Size() + uint32(codec.Uint32(0).Size())
}

type TxElements []*TxElement

func (elems TxElements) Encode() []byte {
	byteset := slice.ParallelAppend(elems, 4, func(i int, _ *TxElement) []byte { return elems[i].Encode() })
	return codec.Byteset(byteset).Encode()
}

func (TxElements) Decode(bytes []byte) TxElements {
	bytesset := codec.Byteset{}.Decode(bytes).(codec.Byteset)
	return slice.ParallelAppend(bytesset, 4, func(i int, _ []byte) *TxElement {
		ele := &TxElement{}
		ele.Decode(bytesset[i])
		return ele
	})
}

func (request *ArbitratorRequest) GobEncode() ([]byte, error) {
	return request.Encode(), nil
}

func (request *ArbitratorRequest) GobDecode(data []byte) error {
	req := request.Decode(data)
	request.TxsListGroup = req.TxsListGroup
	return nil
}

func (request *ArbitratorRequest) Encode() []byte {
	bytes := make([][]byte, len(request.TxsListGroup))
	worker := func(start int, end int, idx int, args ...interface{}) {
		for i := start; i < end; i++ {
			bytes[i] = TxElements(request.TxsListGroup[i]).Encode()
		}
	}
	common.ParallelWorker(len(bytes), 2, worker)
	return codec.Byteset(bytes).Encode()
}

func (ArbitratorRequest) Decode(bytes []byte) *ArbitratorRequest {
	byteset := codec.Byteset{}.Decode(bytes).(codec.Byteset)
	elems := slice.ParallelAppend(byteset, 2, func(i int, _ []byte) []*TxElement {
		return TxElements{}.Decode(byteset[i])
	})
	return &ArbitratorRequest{elems}
}

type arbReq struct {
	Indices []uint32
	Hashes  []byte
	Batches []uint32
}

type arbReqEncoder struct {
	indexBuf []uint32
	hashBuf  []byte
	batchBuf []uint32
}

func newArbReqEncoder() *arbReqEncoder {
	maxSize := 500000
	return &arbReqEncoder{
		indexBuf: make([]uint32, maxSize*2),
		hashBuf:  make([]byte, maxSize*2*32),
		batchBuf: make([]uint32, maxSize*2),
	}
}

func (this *arbReqEncoder) Encode(r *ArbitratorRequest) *arbReq {
	if len(r.TxsListGroup) == 0 {
		return &arbReq{}
	}

	indexOffset := uint32(0)
	dataOffset := 0
	batchOffset := uint32(0)

	prevGroupSize := len(r.TxsListGroup[0])
	count := 1
	for _, elem := range r.TxsListGroup[0] {
		dataOffset += copy(this.hashBuf[dataOffset:], elem.TxHash.Bytes())
		this.batchBuf[batchOffset] = uint32(elem.Batchid)
		batchOffset++
	}

	for i := 1; i < len(r.TxsListGroup); i++ {
		if len(r.TxsListGroup[i]) != prevGroupSize {
			this.indexBuf[indexOffset] = uint32(prevGroupSize)
			this.indexBuf[indexOffset+1] = uint32(count)
			indexOffset += 2
			prevGroupSize = len(r.TxsListGroup[i])
			count = 1
		} else {
			count++
		}

		for _, elem := range r.TxsListGroup[i] {
			dataOffset += copy(this.hashBuf[dataOffset:], elem.TxHash.Bytes())
			this.batchBuf[batchOffset] = uint32(elem.Batchid)
			batchOffset++
		}
	}

	this.indexBuf[indexOffset] = uint32(prevGroupSize)
	this.indexBuf[indexOffset+1] = uint32(count)
	indexOffset += 2

	return &arbReq{
		Indices: this.indexBuf[:indexOffset],
		Hashes:  this.hashBuf[:dataOffset],
		Batches: this.batchBuf[:batchOffset],
	}
}

type arbReqDecoder struct {
	list [][]*TxElement
}

func newArbReqDecoder() *arbReqDecoder {
	list := make([][]*TxElement, 500000)
	for i := range list {
		list[i] = make([]*TxElement, 0, 8)
	}
	return &arbReqDecoder{
		list: list,
	}
}

func (this *arbReqDecoder) Decode(r *arbReq) *ArbitratorRequest {
	offset := 0
	hashOffset := 0
	batchOffset := 0
	for i := 0; i < len(r.Indices); i += 2 {
		subListSize := r.Indices[i]
		count := r.Indices[i+1]
		for j := uint32(0); j < count; j++ {
			this.list[offset] = this.list[offset][:0]
			for k := uint32(0); k < subListSize; k++ {
				hash := ethCommon.BytesToHash(r.Hashes[hashOffset : hashOffset+32])
				this.list[offset] = append(this.list[offset], &TxElement{
					TxHash:  &hash,
					Batchid: uint64(r.Batches[batchOffset]),
				})
				hashOffset += 32
				batchOffset++
			}
			offset++
		}
	}
	return &ArbitratorRequest{
		TxsListGroup: this.list[:offset],
	}
}
