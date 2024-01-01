/*
 *   Copyright (c) 2023 Arcology Network

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

package querycache

import (
	"errors"

	"github.com/hashicorp/go-memdb"
	hashicorpmemdb "github.com/hashicorp/go-memdb"
)

type Reader interface {
	Get(string) (interface{}, error) // dbReader is used to query the database.
}

// QueryableCache is an in-memory database that supports querying.
// It can be used to cache the data of the database to improve the performance of the database by
// reducing the number of queries to the database.
type QueryableCache struct {
	cache       *hashicorpmemdb.MemDB
	size        int               // The number of batch to be cached.
	primaryKeys map[string]string // The primary keys of the tables.
	dbReader    interface {
		Get(string) (interface{}, error) // dbReader is used to query the database.
	}
}

func NewIndex(fieldName string, isPrimaryKey, Unique bool, T any) *hashicorpmemdb.IndexSchema {
	var indexer hashicorpmemdb.Indexer
	switch T.(type) {
	case *string:
		indexer = &memdb.StringFieldIndex{Field: fieldName}
	case *uint64:
		indexer = &memdb.UintFieldIndex{Field: fieldName}
	case *int:
		indexer = &memdb.IntFieldIndex{Field: fieldName}
	case *bool:
		indexer = &memdb.BoolFieldIndex{Field: fieldName}
	default:
		indexer = T.(hashicorpmemdb.Indexer)
	}

	if isPrimaryKey {
		return &memdb.IndexSchema{
			Name:    "id",
			Unique:  true, // Primary keys must be unique.
			Indexer: indexer,
		}
	}

	return &memdb.IndexSchema{
		Name:    fieldName,
		Unique:  Unique,
		Indexer: indexer,
	}
}

func NewTable(name string, indexers ...*hashicorpmemdb.IndexSchema) *hashicorpmemdb.TableSchema {
	table := &hashicorpmemdb.TableSchema{
		Name:    name,
		Indexes: map[string]*hashicorpmemdb.IndexSchema{},
	}

	for i := range indexers {
		table.Indexes[indexers[i].Name] = indexers[i]
	}
	return table
}

func NewQueryableCache(dbReader interface{}, tables ...*hashicorpmemdb.TableSchema) (*QueryableCache, error) {
	schema := &memdb.DBSchema{Tables: map[string]*memdb.TableSchema{}}
	for _, table := range tables {
		schema.Tables[table.Name] = table
	}

	cache, err := memdb.NewMemDB(schema)
	if err != nil {
		return nil, err
	}

	var reader Reader
	if dbReader != nil {
		reader = dbReader.(Reader)
	}

	return &QueryableCache{
		cache:    cache,
		dbReader: reader,
	}, nil
}

func (this *QueryableCache) InsertTable(name string, indexers ...*hashicorpmemdb.IndexSchema) error {
	if this.cache.DBSchema().Tables[name] != nil {
		return errors.New("Table already exists")
	}

	table := &hashicorpmemdb.TableSchema{
		Name:    name,
		Indexes: map[string]*hashicorpmemdb.IndexSchema{},
	}

	for i := range indexers {
		table.Indexes[indexers[i].Name] = indexers[i]
	}
	this.cache.DBSchema().Tables[table.Name] = table
	return nil
}

// Inserts multiple objects into the database. The size of the cache will be increase by 1.
func (this *QueryableCache) Add(table string, args ...interface{}) error {
	txn := this.cache.Txn(true)
	for _, arg := range args {
		if err := txn.Insert(table, arg); err != nil {
			panic(err)
		}
	}
	txn.Commit()
	return nil
}

func (this *QueryableCache) Remove(table string, args ...interface{}) error {
	tx := this.cache.Txn(true)
	for _, arg := range args {
		if err := tx.Delete(table, arg); err != nil {
			return err
		}
	}
	tx.Commit()
	return nil
}

func (this *QueryableCache) Search(table, column string, arg interface{},
	traverse func(table, index string, args ...interface{}) (hashicorpmemdb.ResultIterator, error)) ([]interface{}, error) {
	buffer := make([]interface{}, 0)
	if ok, err := this.IsAcceptableNumeric(arg); !ok {
		return nil, err
	}

	it, err := traverse(table, column, arg)
	if err != nil {
		return nil, err
	}

	for obj := it.Next(); obj != nil; obj = it.Next() {
		buffer = append(buffer, obj)
	}
	return buffer, nil
}

// Check if the arg is an acceptable numeric type.
func (this *QueryableCache) IsAcceptableNumeric(arg interface{}) (bool, error) {
	_, _0 := arg.(uint64)
	_, _1 := arg.(int)
	if !_0 && !_1 {
		return false, errors.New("arg must be uint64 or int")
	}
	return true, nil
}

func (this *QueryableCache) FindFirst(table, column string, arg interface{}) (interface{}, error) {
	return this.cache.Txn(false).First(table, column, arg)
}

func (this *QueryableCache) FindAll(table, column string, arg interface{}) ([]interface{}, error) {
	iter, err := this.cache.Txn(false).Get(table, column, arg)
	if err != nil {
		return nil, err
	}

	buffer := make([]interface{}, 0)
	for obj := iter.Next(); obj != nil; obj = iter.Next() {
		buffer = append(buffer, obj)
	}
	return buffer, nil
}

func (this *QueryableCache) FindGreaterThan(table, column string, arg interface{}) ([]interface{}, error) {
	return this.Search(table, column, arg, this.cache.Txn(false).LowerBound)
}

func (this *QueryableCache) FindLessThan(table, column string, arg interface{}) ([]interface{}, error) {
	return this.Search(table, column, arg, this.cache.Txn(false).ReverseLowerBound)
}
