package storage

import (
	"io/ioutil"
	"os"
)

type SimpleFileDB struct {
	root string
}

func NewSimpleFileDB(root string) *SimpleFileDB {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		os.Mkdir(root, 0755)
	}

	return &SimpleFileDB{
		root: root,
	}
}

func (db *SimpleFileDB) Set(key string, value []byte) error {
	return ioutil.WriteFile(db.root+key, value, os.FileMode(0600))
}

func (db *SimpleFileDB) Get(key string) ([]byte, error) {
	return ioutil.ReadFile(db.root + key)
}

func (db *SimpleFileDB) Delete(key string) error {
	return os.Remove(db.root + key)
}
