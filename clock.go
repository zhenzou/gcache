package gcache

import (
	"sync"
	"time"
)

type clock interface {
	Now() time.Time
}

type realClock struct{}

func newRealClock() clock {
	return realClock{}
}

func (rc realClock) Now() time.Time {
	t := time.Now()
	return t
}

type fakeClock interface {
	clock

	Advance(d time.Duration)
}

func newFakeClock() fakeClock {
	return &fakeclock{
		// Taken from github.com/jonboulle/clockwork: use a fixture that does not fulfill Time.IsZero()
		now: time.Date(1984, time.April, 4, 0, 0, 0, 0, time.UTC),
	}
}

type fakeclock struct {
	now time.Time

	mutex sync.RWMutex
}

func (fc *fakeclock) Now() time.Time {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	t := fc.now
	return t
}

func (fc *fakeclock) Advance(d time.Duration) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.now = fc.now.Add(d)
}
