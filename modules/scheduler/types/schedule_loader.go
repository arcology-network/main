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
	"encoding/hex"
	"fmt"

	scheduler "github.com/arcology-network/eu/new-scheduler"
)

type ScheduleLoader struct {
	scheduler *scheduler.Scheduler
}

func NewScheduleLoader(scheduler *scheduler.Scheduler) *ScheduleLoader {
	return &ScheduleLoader{
		scheduler: scheduler,
	}
}

func (sl *ScheduleLoader) Init(conflictFile string) string {
	logs := ""
	conflictList, err := LoadingConf(conflictFile)
	if err != nil {
		logs += fmt.Sprintf("loading conf err=%v\n", err)
		return logs
	}

	for _, item := range conflictList {
		sl.scheduler.Add([20]byte(stringToBytes(item.LeftAddr)), [4]byte(stringToBytes(item.LeftSign)), [20]byte(stringToBytes(item.RightAddr)), [4]byte(stringToBytes(item.RightSign)))
	}
	return ""
}

func stringToBytes(input string) []byte {
	ret, err := hex.DecodeString(input)
	if err != nil {
		return []byte{}
	}
	return ret
}
