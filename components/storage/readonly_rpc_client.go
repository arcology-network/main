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
