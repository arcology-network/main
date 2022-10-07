package storage

type DataStoreMock struct {
	db map[string]interface{}
}

func NewDataStoreMock() *DataStoreMock {
	return &DataStoreMock{db: make(map[string]interface{})}
}

func (mock *DataStoreMock) Inject(key string, value interface{}) {
	mock.db[key] = value
}

func (mock *DataStoreMock) Retrive(string) (interface{}, error) {
	return nil, nil
}

func (mock *DataStoreMock) BatchRetrive([]string) []interface{} {
	return nil
}

func (mock *DataStoreMock) Precommit() ([]string, interface{}) {
	return nil, nil
}

func (mock *DataStoreMock) Commit() error {
	return nil
}

func (mock *DataStoreMock) UpdateCacheStats([]string, []interface{}) {}

func (mock *DataStoreMock) Dump() ([]string, []interface{}) {
	return nil, nil
}

func (mock *DataStoreMock) Checksum() [32]byte {
	return [32]byte{}
}

func (mock *DataStoreMock) Clear() {}

func (mock *DataStoreMock) Print() {}

func (mock *DataStoreMock) CheckSum() [32]byte {
	return [32]byte{}
}

func (mock *DataStoreMock) Query(string, func(string, string) bool) ([]string, [][]byte, error) {
	return nil, nil, nil
}
