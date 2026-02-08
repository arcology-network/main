package query

// Scheduler 只负责一件事：
// 在未来某个时间点，执行 callback（一次）
type Scheduler interface {
	ScheduleAfter(delay uint64, cb func())
}

type MockScheduler struct {
	callbacks []func()
}

func (s *MockScheduler) ScheduleAfter(delay uint64, cb func()) {
	s.callbacks = append(s.callbacks, cb)
}

// 手动触发所有定时事件
func (s *MockScheduler) FireAll() {
	for _, cb := range s.callbacks {
		cb()
	}
	s.callbacks = nil
}
