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
