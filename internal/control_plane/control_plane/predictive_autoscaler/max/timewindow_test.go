package max

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestTimedWindowMax(t *testing.T) {
	type entry struct {
		time  time.Time
		value int32
	}

	now := time.Now()

	tests := []struct {
		name   string
		expect int32
		values []entry
	}{{
		name: "single value",
		values: []entry{{
			time:  now,
			value: 5,
		}},
		expect: 5,
	}, {
		name: "two values in same second",
		values: []entry{{
			time:  now,
			value: 6,
		}, {
			time:  now.Add(500 * time.Millisecond),
			value: 5,
		}},
		expect: 6,
	}, {
		name: "two values",
		values: []entry{{
			time:  now,
			value: 5,
		}, {
			time:  now.Add(1 * time.Second),
			value: 8,
		}},
		expect: 8,
	}, {
		name: "time gap",
		values: []entry{{
			time:  now,
			value: 5,
		}, {
			time:  now.Add(6 * time.Second),
			value: 4,
		}},
		expect: 4,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewTimeWindow(5*time.Second, 1*time.Second)

			for _, v := range tt.values {
				m.Record(v.time, v.value)
			}

			if got, want := m.Current(), tt.expect; got != want {
				t.Errorf("Current() = %d, expected %d", got, want)
			}
		})
	}
}

func BenchmarkLargeTimeWindowCreate(b *testing.B) {
	for _, duration := range []time.Duration{5 * time.Minute, 15 * time.Minute, 30 * time.Minute, 45 * time.Minute} {
		b.Run(fmt.Sprintf("duration-%v", duration), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = NewTimeWindow(duration, 1*time.Second)
			}
		})
	}
}

func BenchmarkLargeTimeWindowRecord(b *testing.B) {
	w := NewTimeWindow(45*time.Minute, 1*time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		now = now.Add(1 * time.Second)
		w.Record(now, rand.Int31())
	}
}

func BenchmarkLargeTimeWindowAscendingRecord(b *testing.B) {
	w := NewTimeWindow(45*time.Minute, 1*time.Second)
	now := time.Now()

	for i := 0; i < b.N; i++ {
		now = now.Add(1 * time.Second)
		w.Record(now, int32(i))
	}
}

func BenchmarkLargeTimeWindowDescendingRecord(b *testing.B) {
	for _, duration := range []time.Duration{5, 15, 30, 45} {
		b.Run(fmt.Sprintf("duration-%d-minutes", duration), func(b *testing.B) {
			w := NewTimeWindow(duration*time.Minute, 1*time.Second)
			now := time.Now()

			for i := 0; i < b.N; i++ {
				now = now.Add(1 * time.Second)
				w.Record(now, int32(math.MaxInt32-i))
			}
		})
	}
}
