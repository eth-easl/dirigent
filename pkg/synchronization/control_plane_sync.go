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

package synchronization

import (
	"sync"
)

type SyncStructure[K comparable, V any] interface {
	Lock()
	RLock()
	Unlock()
	RUnlock()

	GetMap() map[K]V

	GetKeys() []K        // Not thread safe
	GetValues() []V      // Not thread safe
	Set(key K, value V)  // Not thread safe
	Get(key K) (V, bool) // Not thread safe
	GetNoCheck(key K) V  // Not thread safe
	Remove(key K)        // Not thread safe
	Present(key K) bool  // Not thread safe

	AtomicSet(key K, value V)
	AtomicGet(key K) (V, bool)
	AtomicGetNoCheck(key K) V
	AtomicRemove(key K)

	Len() int       // Atomic operation
	AtomicLen() int // Atomic operation
}

type Structure[K comparable, V any] struct {
	InternalMap  map[K]V
	internalLock *sync.RWMutex
}

func NewControlPlaneSyncStructure[K comparable, V any]() *Structure[K, V] {
	return &Structure[K, V]{
		InternalMap:  make(map[K]V),
		internalLock: &sync.RWMutex{},
	}
}

func (s *Structure[K, V]) Lock() {
	s.internalLock.Lock()
}

func (s *Structure[K, V]) RLock() {
	s.internalLock.RLock()
}

func (s *Structure[K, V]) Unlock() {
	s.internalLock.Unlock()
}

func (s *Structure[K, V]) RUnlock() {
	s.internalLock.RUnlock()
}

func (s *Structure[K, V]) GetMap() map[K]V {
	return s.InternalMap
}

func (s *Structure[K, V]) Set(key K, value V) {
	s.InternalMap[key] = value
}

func (s *Structure[K, V]) Get(key K) (V, bool) {
	v, ok := s.InternalMap[key]
	return v, ok
}

func (s *Structure[K, V]) GetNoCheck(key K) V {
	v, _ := s.InternalMap[key]
	return v
}

func (s *Structure[K, V]) Remove(key K) {
	delete(s.InternalMap, key)
}

func (s *Structure[K, V]) Present(key K) bool {
	_, ok := s.InternalMap[key]
	return ok
}

func (s *Structure[K, V]) AtomicSet(key K, value V) {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	s.Set(key, value)
}

func (s *Structure[K, V]) AtomicGet(key K) (V, bool) {
	s.internalLock.RLock()
	defer s.internalLock.RUnlock()

	return s.Get(key)
}

func (s *Structure[K, V]) AtomicGetNoCheck(key K) V {
	s.internalLock.RLock()
	defer s.internalLock.RUnlock()

	return s.GetNoCheck(key)
}

func (s *Structure[K, V]) AtomicRemove(key K) {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	s.Remove(key)
}

func (s *Structure[K, V]) Len() int {
	return len(s.InternalMap)
}

func (s *Structure[K, V]) AtomicLen() int {
	s.internalLock.RLock()
	defer s.internalLock.RUnlock()

	return len(s.InternalMap)
}

func (s *Structure[K, V]) GetKeys() []K {
	var result []K
	for k := range s.GetMap() {
		result = append(result, k)
	}

	return result
}

func (s *Structure[K, V]) GetValues() []V {
	var result []V
	for _, v := range s.GetMap() {
		result = append(result, v)
	}

	return result
}
