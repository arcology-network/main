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

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/storage-committer/commutative"
	"github.com/arcology-network/storage-committer/interfaces"
	"github.com/arcology-network/storage-committer/noncommutative"
	"github.com/holiman/uint256"
)

func GetBalance(ds interfaces.ReadOnlyStore, addr string) (*big.Int, error) {
	key := getBalancePath(addr)
	obj, err := ds.Retrive(key, new(commutative.U256))
	if err != nil {
		return nil, err
	}
	if obj == nil || obj == nil {
		return big.NewInt(0), nil
	}

	balance := obj.(*commutative.U256).Value().(uint256.Int)
	// uubalance := uint256.Int(*ubalance) //.ToBig()
	// balance := uubalance.ToBig()
	// return balance, nil
	return (&balance).ToBig(), nil
}
func GetNonce(ds interfaces.ReadOnlyStore, addr string) (uint64, error) {
	obj, err := ds.Retrive(getNoncePath(addr), new(commutative.Uint64))
	if err != nil || obj == nil {
		return 0, err
	}
	nonce := obj.(*commutative.Uint64).Value().(uint64)
	return uint64(nonce), nil
}
func GetCode(ds interfaces.ReadOnlyStore, addr string) ([]byte, error) {
	obj, err := ds.Retrive(getCodePath(addr), new(noncommutative.Bytes))
	if err != nil || obj == nil {
		return []byte{}, err
	}
	bys := obj.(*noncommutative.Bytes).Value().(codec.Bytes)
	return []byte(bys), nil
}

func GetStorage(ds interfaces.ReadOnlyStore, addr, key string) ([]byte, error) {
	path := getStorageKeyPath(addr, key)
	obj, err := ds.Retrive(path, new(noncommutative.Bytes))
	if err != nil || obj == nil {
		return []byte{}, err
	}

	bys := obj.(*noncommutative.Bytes).Value().(codec.Bytes)
	return []byte(bys), nil
}
