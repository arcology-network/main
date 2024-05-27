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
	"reflect"
	"testing"

	mtypes "github.com/arcology-network/main/types"
)

func TestExectime(t *testing.T) {
	cache := NewExectimeCaches(2)
	exectime1 := mtypes.StatisticalInformation{
		Key:      "exectime",
		Value:    "ok",
		TimeUsed: 100,
	}
	cache.Save(1, &exectime1)
	exectime2 := mtypes.StatisticalInformation{
		Key:      "exectime",
		Value:    "ok",
		TimeUsed: 200,
	}
	cache.Save(2, &exectime2)

	queryResult := cache.Query(1)
	if !reflect.DeepEqual(*queryResult, exectime1) {
		t.Error("cache save get Error")
		return
	}
	exectime3 := mtypes.StatisticalInformation{
		Key:      "exectime",
		Value:    "ok",
		TimeUsed: 300,
	}
	cache.Save(3, &exectime3)

	queryResult = cache.Query(1)
	if queryResult != nil {
		t.Error("cache remove Error")
		return
	}

	queryResult = cache.Query(2)
	if !reflect.DeepEqual(*queryResult, exectime2) {
		t.Error("cache save get Error")
		return
	}

}
