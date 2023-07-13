package atomic_map

import (
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type AtomicMap[T any] struct {
	m sync.Map
}

func NewAtomicMap[T any]() AtomicMap[T] {
	return AtomicMap[T]{}
}

func (c *AtomicMap[T]) AtomicGet(key T) int64 {
	count, ok := c.m.Load(key)
	if ok {
		val, ok := count.(*int64)
		if !ok {
			logrus.Fatal("Type assertion failed")
		}

		return atomic.LoadInt64(val)
	}

	return 0
}

func (c *AtomicMap[T]) AtomicAdd(key T, value int64) int64 {
	count, loaded := c.m.LoadOrStore(key, &value)
	if loaded {
		val, ok := count.(*int64)
		if !ok {
			logrus.Fatal("Type assertion failed")
		}

		return atomic.AddInt64(val, value)
	}

	val, ok := count.(*int64)
	if !ok {
		logrus.Fatal("Type assertion failed")
	}

	return *val
}

func (c *AtomicMap[T]) AtomicIncrement(key T) {
	c.AtomicAdd(key, 1)
}

func (c *AtomicMap[T]) AtomicDecrement(key T) {
	c.AtomicAdd(key, -1)
}

func (c *AtomicMap[T]) AtomicDelete(key T) {
	c.m.Delete(key)
}
