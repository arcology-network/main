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
