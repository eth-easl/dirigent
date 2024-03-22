package atomic_map

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type AtomicMap[K, V any] struct {
	sync.Map
}

func NewAtomicMap[K, V any]() *AtomicMap[K, V] {
	return &AtomicMap[K, V]{}
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

func (c *AtomicMap[K, V]) GetOrSet(key K, value V) (V, bool) {
	val, loaded := c.LoadOrStore(key, value)

	typedVal, ok := val.(V)
	if !ok {
		logrus.Fatal("Type assertion failed")
	}

	return typedVal, loaded
}

func (c *AtomicMap[K, V]) GetUnsafe(key K) V {
	val, _ := c.Get(key)
	return val
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

func (c *AtomicMap[K, V]) Values() []V {
	output := make([]V, 0)

	c.Range(func(k, v interface{}) bool {
		value, ok := v.(V)
		if !ok {
			logrus.Fatal("error in sync.map")
		}

		output = append(output, value)

		return true
	})

	return output
}

func (c *AtomicMap[K, V]) KeyValues() ([]K, []V) {
	outputKey := make([]K, 0)
	outputValue := make([]V, 0)

	c.Range(func(k, v interface{}) bool {
		key, ok := k.(K)
		if !ok {
			logrus.Fatal("error in sync.map")
		}

		value, ok := v.(V)
		if !ok {
			logrus.Fatal("error in sync.map")
		}

		outputKey = append(outputKey, key)
		outputValue = append(outputValue, value)

		return true
	})

	return outputKey, outputValue
}
