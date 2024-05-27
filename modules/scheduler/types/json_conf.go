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
	"encoding/json"
	"os"
)

type ConflictItem struct {
	LeftAddr  string
	RightAddr string
	LeftSign  string
	RightSign string
}

func LoadingConf(file string) ([]ConflictItem, error) {
	items := []ConflictItem{}
	filePtr, err := os.Open(file)
	if err != nil {
		return []ConflictItem{}, err
	}
	defer filePtr.Close()

	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&items)
	if err != nil {
		//fmt.Println("Decoder failed", err.Error())
		return items, err
	}
	return items, nil
}
