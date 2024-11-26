/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
