package atomic_map

import (
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type AtomicMapCounter[K any] struct {
	m sync.Map
}

func NewAtomicMapCounter[K any]() AtomicMapCounter[K] {
	return AtomicMapCounter[K]{}
}

func (c *AtomicMapCounter[K]) Get(key K) int64 {
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

func (c *AtomicMapCounter[K]) AtomicAdd(key K, value int64) int64 {
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

func (c *AtomicMapCounter[K]) AtomicIncrement(key K) {
	c.AtomicAdd(key, 1)
}

func (c *AtomicMapCounter[K]) AtomicDecrement(key K) {
	c.AtomicAdd(key, -1)
}

func (c *AtomicMapCounter[K]) Delete(key K) {
	c.m.Delete(key)
}
