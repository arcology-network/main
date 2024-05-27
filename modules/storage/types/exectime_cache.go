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
	"fmt"

	mtypes "github.com/arcology-network/main/types"
)

type ExectimeCaches struct {
	caches *DataCache
}

func NewExectimeCaches(cache int) *ExectimeCaches {
	return &ExectimeCaches{
		caches: NewDataCache(cache),
	}
}
func (ec *ExectimeCaches) GetKey(height uint64) string {
	return fmt.Sprintf("statisticalInformation-%v", height)
}

func (ec *ExectimeCaches) Query(height uint64) *mtypes.StatisticalInformation {
	key := ec.GetKey(height)
	staticalInfo := ec.caches.Query(key)
	if staticalInfo != nil {
		return staticalInfo.(*mtypes.StatisticalInformation)
	}
	return nil
}

func (ec *ExectimeCaches) Save(height uint64, staticalInfo *mtypes.StatisticalInformation) {
	key := ec.GetKey(height)
	ec.caches.Add(height, []string{key}, []interface{}{staticalInfo})
}
