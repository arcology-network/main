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

package config

import "encoding/json"

const (
	WorkModePassive = 1
	WorkModeActive  = 2
)

type PeerConfig struct {
	ID              string `json:"id"`
	Host            string `json:"host"`
	Port            int    `json:"port"`
	ConnectionCount byte   `json:"connections"`
	AssignTo        string `json:"assign_to"`
	Mode            int    `json:"mode"`
}

func (cfg PeerConfig) ToJsonStr() (string, error) {
	b, err := json.Marshal(cfg)
	return string(b), err
}

func (cfg *PeerConfig) FromJsonStr(str string) error {
	return json.Unmarshal([]byte(str), cfg)
}
