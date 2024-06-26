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
	"math/big"
	"reflect"
	"testing"

	mtypes "github.com/arcology-network/main/types"
	evm "github.com/ethereum/go-ethereum"
	evmCommon "github.com/ethereum/go-ethereum/common"
	evmTypes "github.com/ethereum/go-ethereum/core/types"
)

func TestFilter(t *testing.T) {
	filetrs := [][]evmCommon.Hash{}
	topics := []evmCommon.Hash{
		evmCommon.HexToHash("66812f708302896954212d553f4cf11dc8859e32cb3ae1decfd955aba6718312"),
		evmCommon.HexToHash("714222414d021b26175afe55364239d7c12a8eb54e300b4256402bffea20da74"),
		evmCommon.HexToHash("927cfa6decccd60f650e6566dc319240299ce3c08ff4afa1f75721e016407b57"),
	}

	if !reflect.DeepEqual(true, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("{} filter  Error")
	}

	if !reflect.DeepEqual(true, mtypes.FiltereTopic(nil, topics)) {
		t.Error("nil filter  Error")
	}

	filetrs = [][]evmCommon.Hash{{evmCommon.HexToHash("66812f708302896954212d553f4cf11dc8859e32cb3ae1decfd955aba6718312")}}

	if !reflect.DeepEqual(true, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("{{A}} filter  Error")
	}
	filetrs = [][]evmCommon.Hash{{}, {evmCommon.HexToHash("714222414d021b26175afe55364239d7c12a8eb54e300b4256402bffea20da74")}}

	if !reflect.DeepEqual(true, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("{{}, {B}} filter  Error")
	}

	filetrs = [][]evmCommon.Hash{
		{evmCommon.HexToHash("66812f708302896954212d553f4cf11dc8859e32cb3ae1decfd955aba6718312")},
		{evmCommon.HexToHash("714222414d021b26175afe55364239d7c12a8eb54e300b4256402bffea20da74")},
	}

	if !reflect.DeepEqual(true, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("{{A},{B}} filter  Error")
	}

	filetrs = [][]evmCommon.Hash{
		{
			evmCommon.HexToHash("66812f708302896954212d553f4cf11dc8859e32cb3ae1decfd955aba6718312"),
			evmCommon.HexToHash("948bb02cc82c831825e543a3a161f092c0f4e6826b7ca8fe953b709ba86c8340"),
		},
		{
			evmCommon.HexToHash("6ec95c015efb41f1212f06f0e2c630a1a2a521fdf79ad1391c3bc55fb68959dc"),
			evmCommon.HexToHash("714222414d021b26175afe55364239d7c12a8eb54e300b4256402bffea20da74"),
		},
	}
	if !reflect.DeepEqual(true, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("{{A, B}, {C, D}} filter  Error")
	}

	filetrs = [][]evmCommon.Hash{
		{evmCommon.HexToHash("66812f708302896954212d553f4cf11dc8859e32cb3ae1decfd955aba6718312")},
		{evmCommon.HexToHash("714222414d021b26175afe55364239d7c12a8eb54e300b4256402bffea20da74")},
		{evmCommon.HexToHash("927cfa6decccd60f650e6566dc319240299ce3c08ff4afa1f75721e016407b57")},
		{},
	}
	if !reflect.DeepEqual(false, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("filetrs size > topics size filter  Error")
	}

	filetrs = [][]evmCommon.Hash{
		{evmCommon.HexToHash("66812f708302896954212d553f4cf11dc8859e32cb3ae1decfd955aba6718312")},
		{evmCommon.HexToHash("714222414d021b26175afe55364239d7c12a8eb54e300b4256402bffea20da74")},
		{evmCommon.HexToHash("927cfa6decccd60f650e6566dc319240299ce3c08ff4afa1f75721e016407b57")},
	}
	if !reflect.DeepEqual(true, mtypes.FiltereTopic(filetrs, topics)) {
		t.Error("{{A},{B},{C}} filter  Error")
	}
}

func TestCache(t *testing.T) {
	hashes := []evmCommon.Hash{
		evmCommon.HexToHash("65cebc3290e9d0b1b829c52ab29f56644da2ec2d40dffa8cc6cbb6f075b64868"),
		evmCommon.HexToHash("046adbeb350177eb24729e06d588683c58fd4d47efe63c86c8481b1e31bef9eb"),
		evmCommon.HexToHash("30f9990ac8ba7fc6524349f5d04e164ad4ea1332da03cb6120b8245bb3111271"),
		evmCommon.HexToHash("ba0854ad05fe15aedda700358d9a1828476569d832cd27fe1e1dca3396129363"),

		evmCommon.HexToHash("19b77851c540b06f56082f1dae419c7a9935aabe93bbfb05fd5e098ea61ed0bc"),
		evmCommon.HexToHash("809f675444f2123543c53dcdc2e1b0521639efe85d8f5bb8da07163ce1193ae8"),
		evmCommon.HexToHash("d2f2f030885c01963b0ebf34e07b683bdea3645deb1741012d3ae8afd6fef122"),
		evmCommon.HexToHash("6e922066483291034dbbf02caf27333f14b781f3ef33fb242b05dbaef50ea8fc"),

		evmCommon.HexToHash("5d5291e4d3b17905c7cea61458eec5b3014951715d8916088823ba99f7d27b31"),
		evmCommon.HexToHash("ceadb0381fae28438b66787027611361bf734571e6c176dbb0d1ae23a9256b3c"),
		evmCommon.HexToHash("1779d7cbac346c4dd4fe45b7230afe059c26259da4ccf4a3792e1640a180caac"),
		evmCommon.HexToHash("a81271460d441189fc8bf152cd31cc9c65eaeea1bf7ec4a091d3908814549ff7"),

		evmCommon.HexToHash("8e7b805c11007230f11d1cad66e0a2ca6b0d992fa0a5a87158c4574b0192eec5"),
		evmCommon.HexToHash("18a6df993d58d2de7134cf22a36af7de5d1481384281f1710e0c99bde3c5fe23"),
		evmCommon.HexToHash("a33c7671a6a4d29f6aaf950a2ba0a810c8f01325dc39e0ba96a77061c155eca7"),
		evmCommon.HexToHash("5f29535942e5ef712a6a09f011634bb077681ddc444c0f26b9e20f015c6edfa5"),
	}
	addres := []evmCommon.Address{
		evmCommon.HexToAddress("0xcb5223CED9dB576B666E1CB7936B7633B82898e0"),
		evmCommon.HexToAddress("0x56D4271067BA9dE8739908DEc9b437D48953288a"),
		evmCommon.HexToAddress("0xA5D87b9756185d498136cE7009523C43F6a6D8D1"),
		evmCommon.HexToAddress("0x80d906d5a1d853EcB8ee1Ee0DDe362D8C182ddE3"),

		evmCommon.HexToAddress("0x0F118c0979C4228c64e19f6339f81B72f27D1A6d"),
		evmCommon.HexToAddress("0x313a5a5DB1D97Dfe98d8e411Ff732f8F5C391b55"),
		evmCommon.HexToAddress("0x7E12b44DfC4a411BBD8FC969727b297B92Bf1F9d"),
		evmCommon.HexToAddress("0x0f946C307D98104e8aC61129D631a36562901345"),
	}
	evmlogs := []*evmTypes.Log{
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[0].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[1].Bytes()),
				evmCommon.BytesToHash(hashes[2].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[1].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[3].Bytes()),
				evmCommon.BytesToHash(hashes[4].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[2].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[5].Bytes()),
				evmCommon.BytesToHash(hashes[6].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[3].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[7].Bytes()),
				evmCommon.BytesToHash(hashes[8].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[4].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[9].Bytes()),
				evmCommon.BytesToHash(hashes[10].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[5].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[11].Bytes()),
				evmCommon.BytesToHash(hashes[12].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[1].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[3].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[11].Bytes()),
				evmCommon.BytesToHash(hashes[12].Bytes()),
			},
		},
		&evmTypes.Log{
			BlockHash: evmCommon.BytesToHash(hashes[1].Bytes()),
			Address:   evmCommon.BytesToAddress(addres[3].Bytes()),
			Topics: []evmCommon.Hash{
				evmCommon.BytesToHash(hashes[13].Bytes()),
				evmCommon.BytesToHash(hashes[14].Bytes()),
			},
		},
	}

	logcache := NewLogCaches(2)
	receipts := []*evmTypes.Receipt{
		&evmTypes.Receipt{
			BlockHash: evmCommon.BytesToHash(hashes[2].Bytes()),
			Logs: []*evmTypes.Log{
				&evmTypes.Log{
					BlockHash: hashes[2],
					Address:   addres[4],
					Topics: []evmCommon.Hash{
						hashes[7], hashes[9],
					},
				},
				&evmTypes.Log{
					BlockHash: hashes[2],
					Address:   addres[4],
					Topics: []evmCommon.Hash{
						hashes[8], hashes[3],
					},
				},
			},
		},
	}

	logcache.Add(4, receipts)

	receipts = []*evmTypes.Receipt{
		&evmTypes.Receipt{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Logs: []*evmTypes.Log{
				&evmTypes.Log{
					BlockHash: hashes[0],
					Address:   addres[0],
					Topics: []evmCommon.Hash{
						hashes[1], hashes[2],
					},
				},
				&evmTypes.Log{
					BlockHash: hashes[0],
					Address:   addres[1],
					Topics: []evmCommon.Hash{
						hashes[3], hashes[4],
					},
				},
			},
		},
		&evmTypes.Receipt{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Logs: []*evmTypes.Log{
				&evmTypes.Log{
					BlockHash: hashes[0],
					Address:   addres[2],
					Topics: []evmCommon.Hash{
						hashes[5], hashes[6],
					},
				},
				&evmTypes.Log{
					BlockHash: hashes[0],
					Address:   addres[3],
					Topics: []evmCommon.Hash{
						hashes[7], hashes[8],
					},
				},
			},
		},
		&evmTypes.Receipt{
			BlockHash: evmCommon.BytesToHash(hashes[0].Bytes()),
			Logs: []*evmTypes.Log{
				&evmTypes.Log{
					BlockHash: hashes[0],
					Address:   addres[4],
					Topics: []evmCommon.Hash{
						hashes[9], hashes[10],
					},
				},
				&evmTypes.Log{
					BlockHash: hashes[0],
					Address:   addres[5],
					Topics: []evmCommon.Hash{
						hashes[11], hashes[12],
					},
				},
			},
		},
	}

	logcache.Add(5, receipts)
	hash := evmCommon.BytesToHash(hashes[0].Bytes())
	query := evm.FilterQuery{
		BlockHash: &hash,
	}
	logs := logcache.Query(query)
	if !reflect.DeepEqual(evmlogs[0:6], logs) {
		t.Error("blockhash query  Error")
	}

	query = evm.FilterQuery{
		//BlockHash: &hash,
		Addresses: []evmCommon.Address{
			evmCommon.BytesToAddress(addres[0].Bytes()),
			evmCommon.BytesToAddress(addres[3].Bytes()),
		},
	}
	filteredLogs := []*evmTypes.Log{evmlogs[0], evmlogs[3]}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("address query  Error")
	}

	hash = evmCommon.BytesToHash(hashes[0].Bytes())
	query = evm.FilterQuery{
		BlockHash: &hash,
		Addresses: []evmCommon.Address{
			evmCommon.BytesToAddress(addres[0].Bytes()),
			evmCommon.BytesToAddress(addres[3].Bytes()),
		},
	}
	filteredLogs = []*evmTypes.Log{evmlogs[0], evmlogs[3]}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("blockhash and address query  Error")
	}

	query = evm.FilterQuery{
		Topics: [][]evmCommon.Hash{
			{evmCommon.BytesToHash(hashes[1].Bytes()), evmCommon.BytesToHash(hashes[5].Bytes())},
		},
	}
	filteredLogs = []*evmTypes.Log{evmlogs[0], evmlogs[2]}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("topics {{A,B}} query  Error")
	}

	query = evm.FilterQuery{
		Topics: [][]evmCommon.Hash{
			{evmCommon.BytesToHash(hashes[5].Bytes())},
			{evmCommon.BytesToHash(hashes[6].Bytes())},
		},
	}
	filteredLogs = []*evmTypes.Log{evmlogs[2]}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("topics {{A},{B}} query  Error")
	}

	query = evm.FilterQuery{
		BlockHash: &hash,
		Addresses: []evmCommon.Address{
			evmCommon.BytesToAddress(addres[0].Bytes()),
			evmCommon.BytesToAddress(addres[3].Bytes()),
		},
		Topics: [][]evmCommon.Hash{
			{evmCommon.BytesToHash(hashes[7].Bytes())},
		},
	}
	filteredLogs = []*evmTypes.Log{evmlogs[3]}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("blockhash address and topics query  Error")
	}

	receipts = []*evmTypes.Receipt{
		&evmTypes.Receipt{
			BlockHash: evmCommon.BytesToHash(hashes[1].Bytes()),
			Logs: []*evmTypes.Log{
				&evmTypes.Log{
					BlockHash: hashes[1],
					Address:   addres[3],
					Topics: []evmCommon.Hash{
						hashes[11], hashes[12],
					},
				},
				&evmTypes.Log{
					BlockHash: hashes[1],
					Address:   addres[3],
					Topics: []evmCommon.Hash{
						hashes[13], hashes[14],
					},
				},
			},
		},
	}

	logcache.Add(6, receipts)
	if !reflect.DeepEqual(2, len(logcache.Caches)) {
		t.Error("remove caches query  Error")
	}
	if !reflect.DeepEqual(uint64(6), logcache.LatestHeight) {
		t.Error("latest height  Error")
	}
	query = evm.FilterQuery{
		FromBlock: big.NewInt(7),
	}
	filteredLogs = []*evmTypes.Log{}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("from query  Error")
	}

	query = evm.FilterQuery{
		FromBlock: big.NewInt(4),
		ToBlock:   big.NewInt(7),
	}
	//filteredLogs = []*evmTypes.Log{}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(evmlogs, logs) {
		t.Error("from to query  Error")
	}

	query = evm.FilterQuery{
		FromBlock: big.NewInt(4),
		ToBlock:   big.NewInt(7),
		BlockHash: &hash,
		Addresses: []evmCommon.Address{
			evmCommon.BytesToAddress(addres[0].Bytes()),
			evmCommon.BytesToAddress(addres[3].Bytes()),
		},
		Topics: [][]evmCommon.Hash{
			{evmCommon.BytesToHash(hashes[7].Bytes())},
		},
	}
	filteredLogs = []*evmTypes.Log{evmlogs[3]}
	logs = logcache.Query(query)
	if !reflect.DeepEqual(filteredLogs, logs) {
		t.Error("from to blockhash address and topics query  Error")
	}
}
