package atomic_map

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type AtomicMap[K, V any] struct {
	sync.Map
}

func NewAtomicMap[K, V any]() AtomicMap[K, V] {
	return AtomicMap[K, V]{}
}

func (c *AtomicMap[K, V]) Find(key K) bool {
	_, ok := c.Load(key)
	return ok
}

func (c *AtomicMap[K, V]) Get(key K) (V, bool) {
	count, ok := c.Load(key)
	if !ok {
		var genericValue V
		return genericValue, false
	}

	val, ok := count.(V)
	if !ok {
		logrus.Fatal("Type assertion failed")
	}

	return val, true
}

func (c *AtomicMap[K, V]) Set(key K, value V) {
	c.Store(key, value)
}

func (c *AtomicMap[K, V]) RemoveKey(key K) {
	c.Delete(key)
}

func (c *AtomicMap[K, V]) Len() int {
	return len(c.Keys())
}

func (c *AtomicMap[K, V]) Keys() []K {
	output := make([]K, 0)

	c.Range(func(k, v interface{}) bool {
		value, ok := k.(K)
		if !ok {
			logrus.Fatal("error in sync.map")
		}

		output = append(output, value)

		return true
	})

	return output
}
