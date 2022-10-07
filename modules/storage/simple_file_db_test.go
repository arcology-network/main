package storage

import (
	"bytes"
	"testing"
)

func TestGetSet(t *testing.T) {
	db := NewSimpleFileDB("./testdb/")
	key := "key"
	value := []byte{1, 2, 3, 4, 5}

	db.Set(key, value)
	if v, err := db.Get(key); err != nil {
		t.Error("error", err)
	} else if !bytes.Equal(v, value) {
		t.Error("v", v)
	}
}
