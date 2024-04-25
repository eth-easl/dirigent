package max

import (
	"math"
	"time"
)

// TimeWindow is a descending minima window whose indexes are calculated based
// on time.Time values.
type TimeWindow struct {
	window      *window
	granularity time.Duration
}

// NewTimeWindow creates a new TimeWindow.
func NewTimeWindow(duration, granularity time.Duration) *TimeWindow {
	buckets := int(math.Ceil(float64(duration) / float64(granularity)))
	return &TimeWindow{window: newWindow(buckets), granularity: granularity}
}

// Record records a value in the bucket derived from the given time.
func (t *TimeWindow) Record(now time.Time, value int32) {
	index := int(now.Unix()) / int(t.granularity.Seconds())
	t.window.Record(index, value)
}

// Current returns the current maximum value observed in the previous
// window duration.
func (t *TimeWindow) Current() int32 {
	return t.window.Current()
}
