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
	"github.com/sirupsen/logrus"
	"sync"
	"sync/atomic"
)

type AtomicMapCounter[K any] struct {
	m sync.Map
}

func NewAtomicMapCounter[K any]() *AtomicMapCounter[K] {
	return &AtomicMapCounter[K]{}
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
