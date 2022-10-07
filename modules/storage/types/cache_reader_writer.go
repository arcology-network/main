package types

import (
	"math/big"

	urlcommon "github.com/HPISTechnologies/concurrenturl/v2/common"
	"github.com/HPISTechnologies/concurrenturl/v2/type/commutative"
	"github.com/HPISTechnologies/concurrenturl/v2/type/noncommutative"
)

// func SetValues(ds DB, keys []string, values [][]byte) error {
// 	return ds.BatchSet(keys, values)
// }
// func GetValues(ds DB, keys []string) [][]byte {
// 	values, err := ds.BatchGet(keys)
// 	if err != nil {
// 		return [][]byte{}
// 	}
// 	return values
// }
func GetBalance(ds urlcommon.DatastoreInterface, addr string) (*big.Int, error) {
	key := getBalancePath(addr)
	obj, err := ds.Retrive(key)
	if err != nil {
		return nil, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*commutative.Balance).Value().(*big.Int), nil
}
func GetNonce(ds urlcommon.DatastoreInterface, addr string) (int64, error) {
	obj, err := ds.Retrive(getNoncePath(addr))
	if err != nil {
		return 0, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*commutative.Int64).Value().(int64), nil
}
func GetCode(ds urlcommon.DatastoreInterface, addr string) ([]byte, error) {
	obj, err := ds.Retrive(getCodePath(addr))
	if err != nil {
		return []byte{}, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*noncommutative.Bytes).Data(), nil
}

func GetStorage(ds urlcommon.DatastoreInterface, addr, key string) ([]byte, error) {
	obj, err := ds.Retrive(getStorageKeyPath(addr, key))
	if err != nil {
		return []byte{}, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*noncommutative.Bytes).Data(), nil
}

func GetContainerArray(ds urlcommon.DatastoreInterface, addr, id string, idx int) ([]byte, error) {
	obj, err := ds.Retrive(getContainerArrayPath(addr, id, idx))
	if err != nil {
		return []byte{}, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*noncommutative.Bytes).Data(), nil
}

func GetContainerMap(ds urlcommon.DatastoreInterface, addr, id string, key []byte) ([]byte, error) {
	obj, err := ds.Retrive(getContainerMapPath(addr, id, key))
	if err != nil {
		return []byte{}, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*noncommutative.Bytes).Data(), nil
}

func GetContainerQueue(ds urlcommon.DatastoreInterface, addr, id string, key []byte) ([]byte, error) {
	obj, err := ds.Retrive(getContainerQueuePath(addr, id, key))
	if err != nil {
		return []byte{}, err
	}
	return obj.(urlcommon.TypeInterface).Value().(*noncommutative.Bytes).Data(), nil
}
