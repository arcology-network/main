package types

import (
	"math/big"

	"github.com/arcology-network/common-lib/codec"
	"github.com/arcology-network/concurrenturl/commutative"
	"github.com/arcology-network/concurrenturl/interfaces"
	"github.com/arcology-network/concurrenturl/noncommutative"
	"github.com/holiman/uint256"
)

func GetBalance(ds interfaces.Datastore, addr string) (*big.Int, error) {
	key := getBalancePath(addr)
	obj, err := ds.Retrive(key)
	if err != nil {
		return nil, err
	}
	if obj == nil || obj == nil {
		return big.NewInt(0), nil
	}

	ubalance := obj.(*commutative.U256).Value().(*codec.Uint256)
	uubalance := uint256.Int(*ubalance) //.ToBig()
	balance := uubalance.ToBig()
	return balance, nil

}
func GetNonce(ds interfaces.Datastore, addr string) (uint64, error) {
	obj, err := ds.Retrive(getNoncePath(addr))
	if err != nil || obj == nil {
		return 0, err
	}
	nonce := obj.(*commutative.Uint64).Value().(*codec.Uint64).Get().(codec.Uint64)
	return uint64(nonce), nil
}
func GetCode(ds interfaces.Datastore, addr string) ([]byte, error) {
	obj, err := ds.Retrive(getCodePath(addr))
	if err != nil || obj == nil {
		return []byte{}, err
	}
	bys := obj.(*noncommutative.Bytes).Value().(codec.Bytes)
	return []byte(bys), nil
}

func GetStorage(ds interfaces.Datastore, addr, key string) ([]byte, error) {
	path := getStorageKeyPath(addr, key)
	obj, err := ds.Retrive(path)
	if err != nil || obj == nil {
		return []byte{}, err
	}

	bys := obj.(*noncommutative.Bytes).Value().(codec.Bytes)
	return []byte(bys), nil
}
