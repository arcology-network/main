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

// import (
// 	intf "github.com/arcology-network/streamer/interface"
// )

// type ReadonlyRpcClient struct{}

// func NewReadonlyRpcClient() *ReadonlyRpcClient {
// 	return &ReadonlyRpcClient{}
// }

// func (this *ReadonlyRpcClient) Get(key string) ([]byte, error) {
// 	var values [][]byte
// 	err := intf.Router.Call("urlstore", "Get", &[]string{key}, &values)
// 	if err != nil || len(values) != 1 {
// 		return nil, err
// 	}
// 	return values[0], err
// }

// func (this *ReadonlyRpcClient) BatchGet(keys []string) ([][]byte, error) {
// 	var values [][]byte
// 	err := intf.Router.Call("urlstore", "Get", &keys, &values)
// 	return values, err
// }

// // Ready only, do nothing
// func (*ReadonlyRpcClient) Set(path string, v []byte) error           { return nil }
// func (*ReadonlyRpcClient) BatchSet(paths []string, v [][]byte) error { return nil }

// func (*ReadonlyRpcClient) Query(pattern string, condition func(string, string) bool) ([]string, [][]byte, error) {
// 	var response QueryResponse
// 	err := intf.Router.Call("urlstore", "Query", &pattern, &response)
// 	return response.Keys, response.Values, err
// }

type QueryResponse struct {
	Keys   []string
	Values [][]byte
}
