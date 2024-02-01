package types

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"reflect"
	"testing"
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

func PrepareNewArbitrator() *ArbitratorRequest {
	hashes := []ethCommon.Hash{
		ethCommon.BytesToHash([]byte{1, 2, 3}),
		ethCommon.BytesToHash([]byte{4, 5, 6}),
		ethCommon.BytesToHash([]byte{7, 8, 9}),
		ethCommon.BytesToHash([]byte{10, 11, 12}),
		ethCommon.BytesToHash([]byte{13, 14, 15}),
		ethCommon.BytesToHash([]byte{16, 17, 18}),
		ethCommon.BytesToHash([]byte{19, 20, 21}),
	}
	list := [][]*TxElement{
		{
			{
				TxHash:  &hashes[0],
				Batchid: 1,
				Txid:    11,
			},
		},
		{
			{
				TxHash:  &hashes[1],
				Batchid: 2,
				Txid:    12,
			},
			{
				TxHash:  &hashes[2],
				Batchid: 3,
				Txid:    13,
			},
		},
		{
			{
				TxHash:  &hashes[3],
				Batchid: 4,
				Txid:    14,
			},
			{
				TxHash:  &hashes[4],
				Batchid: 5,
				Txid:    15,
			},
		},
		{
			{
				TxHash:  &hashes[5],
				Batchid: 6,
				Txid:    16,
			},
		},
		{
			{
				TxHash:  &hashes[6],
				Batchid: 7,
			},
		},
	}

	return &ArbitratorRequest{
		TxsListGroup: list,
	}
}

func TestRequestEncodeDecode(t *testing.T) {
	req := PrepareNewArbitrator()
	data, err := req.GobEncode()

	if err != nil {
		fmt.Printf(" Arbitrate.GobEncode err=%v\n", err)
		return

	}

	fmt.Printf(" Arbitrate.GobEncode result=%x\n", data)

	request := ArbitratorRequest{}

	err = request.GobDecode(data)
	if err != nil {
		fmt.Printf(" Arbitrate.GobDecode err=%v\n", err)
		return

	}
	for i := range request.TxsListGroup {
		for j := range request.TxsListGroup[i] {
			fmt.Printf(" Arbitrate.GobDecode element=%v\n", request.TxsListGroup[i][j])
		}
	}
	fmt.Printf(" Arbitrate.GobDecode result=%v\n", request)
}

func TestEncodeDecode(t *testing.T) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(PrepareNewArbitrator())
	var ar ArbitratorRequest
	gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(&ar)

	t.Log(ar.TxsListGroup[0][0])
	t.Log(ar.TxsListGroup[1][0], ar.TxsListGroup[1][1])
	t.Log(ar.TxsListGroup[2][0], ar.TxsListGroup[2][1])
	t.Log(ar.TxsListGroup[3][0])
	t.Log(ar.TxsListGroup[4][0])
}

func BenchmarkArbitratorRequestEncode(b *testing.B) {
	size := 500000
	list := make([][]*TxElement, size)
	for i := 0; i < size; i++ {
		list[i] = []*TxElement{
			{
				TxHash:  &ethCommon.Hash{},
				Batchid: 1,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		encoder := gob.NewEncoder(&buf)
		encoder.Encode(&ArbitratorRequest{
			TxsListGroup: list,
		})
	}
}

func BenchmarkArbitratorRequestDecode(b *testing.B) {
	size := 500000
	list := make([][]*TxElement, size)
	for i := 0; i < size; i++ {
		list[i] = []*TxElement{
			{
				TxHash:  &ethCommon.Hash{},
				Batchid: 1,
			},
		}
	}
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	encoder.Encode(&ArbitratorRequest{
		TxsListGroup: list,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var ar ArbitratorRequest
		gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(&ar)
	}
}

func TestArbitratorRequest(t *testing.T) {
	in := PrepareNewArbitrator()
	bytes := in.Encode()
	out := ArbitratorRequest{}.Decode(bytes)
	if !reflect.DeepEqual(*in, *out) {
		t.Error("Mismatch")
	}
}

func TestArbitratorRequestPerformanec(t *testing.T) {
	t0 := time.Now()
	rows := 1000
	cols := 500
	txInfo := make([][]*TxElement, rows)
	for i := 0; i < len(txInfo); i++ {
		txInfo[i] = []*TxElement{}
		for j := 0; j < cols; j++ {
			hash := ethCommon.BytesToHash([]byte{1})
			txInfo[i] = append(txInfo[i], &TxElement{&hash, uint64(i), uint32(0)})

		}
	}
	fmt.Println("ArbitratorRequest:", time.Now().Sub(t0))

	t0 = time.Now()
	in := ArbitratorRequest{txInfo}
	bytes := in.Encode()
	fmt.Println("in.Encode():", time.Now().Sub(t0))

	t0 = time.Now()
	out := ArbitratorRequest{}.Decode(bytes)
	fmt.Println("in.Decode():", time.Now().Sub(t0))

	t0 = time.Now()
	if !reflect.DeepEqual(in, *out) {
		t.Error("Mismatch")
	}
	fmt.Println("DeepEqual():", time.Now().Sub(t0))
}
