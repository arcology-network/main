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
