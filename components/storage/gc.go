package storage

import (
	"runtime"
	"time"

	"github.com/arcology-network/streamer/actor"
	"github.com/arcology-network/streamer/log"
	"go.uber.org/zap"
)

type Gc struct {
	actor.WorkerThread
}

func NewGc(lanes int, groupid string) actor.IWorkerEx {
	gc := Gc{}
	gc.Set(lanes, groupid)
	return &gc
}

func (gc *Gc) Inputs() ([]string, bool) {
	return []string{actor.MsgGc}, false
}

func (gc *Gc) Outputs() map[string]int {
	return map[string]int{}
}

func (gc *Gc) OnStart() {
}

func (gc *Gc) OnMessageArrived(msgs []*actor.Message) error {
	for _, v := range msgs {
		switch v.Name {
		case actor.MsgGc:
			t := time.Now()
			runtime.GC()
			gc.AddLog(log.LogLevel_Info, "gc completed ---->", zap.Duration("time", time.Since(t)))
		}
	}
	return nil
}
