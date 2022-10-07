package types

import (
	"sync"
)

type DB interface {
	Set(string, []byte) error
	Get(string) ([]byte, error)
	BatchSet([]string, [][]byte) error
	BatchGet([]string) ([][]byte, error)
}

var (
	db       DB
	initOnce sync.Once
)

func CreateDB(params map[string]interface{}) DB {
	initOnce.Do(func() {
		// remotes := params["remote_caches"].(string)
		// if remotes == "" {
		db = NewMemoryDB()
		// } else {
		// 	db = NewRedisDB(strings.Split(remotes, ","))
		// }
	})

	return db
}
