package types

import (
	"math/big"

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
	if obj == nil {
		return big.NewInt(0), nil
	}

	return obj.(*commutative.U256).Value().(*uint256.Int).ToBig(), nil

}
func GetNonce(ds interfaces.Datastore, addr string) (uint64, error) {
	obj, err := ds.Retrive(getNoncePath(addr))
	if err != nil {
		return 0, err
	}
	return obj.(*commutative.Uint64).Value().(uint64), nil
}
func GetCode(ds interfaces.Datastore, addr string) ([]byte, error) {
	obj, err := ds.Retrive(getCodePath(addr))
	if err != nil {
		return []byte{}, err
	}
	return obj.(*noncommutative.Bytes).Value().([]byte), nil
}

func GetStorage(ds interfaces.Datastore, addr, key string) ([]byte, error) {
	obj, err := ds.Retrive(getStorageKeyPath(addr, key))
	if err != nil {
		return []byte{}, err
	}
	return obj.(*noncommutative.Bytes).Value().([]byte), nil
}

func GetContainerArray(ds interfaces.Datastore, addr, id string, idx int) ([]byte, error) {
	obj, err := ds.Retrive(getContainerArrayPath(addr, id, idx))
	if err != nil {
		return []byte{}, err
	}
	return obj.(*noncommutative.Bytes).Value().([]byte), nil
}

func GetContainerMap(ds interfaces.Datastore, addr, id string, key []byte) ([]byte, error) {
	obj, err := ds.Retrive(getContainerMapPath(addr, id, key))
	if err != nil {
		return []byte{}, err
	}
	return obj.(*noncommutative.Bytes).Value().([]byte), nil
}

func GetContainerQueue(ds interfaces.Datastore, addr, id string, key []byte) ([]byte, error) {
	obj, err := ds.Retrive(getContainerQueuePath(addr, id, key))
	if err != nil {
		return []byte{}, err
	}
	return obj.(*noncommutative.Bytes).Value().([]byte), nil
}
