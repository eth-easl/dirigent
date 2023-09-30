package synchronization

import "sync"

type SyncStructure[K comparable, V any] interface {
	SetIfAbsent(key K, value V) bool // Atomic operation
	RemoveIfPresent(key K) bool      // Atomic operation

	Lock()
	RLock()
	Unlock()
	RUnlock()

	GetMap() map[K]V

	Set(key K, value V)  // Not thread safe
	Get(key K) (V, bool) // Not thread safe
	GetNoCheck(key K) V  // Not thread safe
	Remove(key K)        // Not thread safe
	Present(key K) bool  // Not thread safe

	AtomicSet(key K, value V)
	AtomicGet(key K) (V, bool)
	AtomicRemove(key K)

	Len() int // Atomic operation
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

func (s *Structure[K, V]) SetIfAbsent(key K, value V) bool {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	if _, ok := s.InternalMap[key]; !ok {
		s.InternalMap[key] = value
		return true
	}

	return false
}

func (s *Structure[K, V]) RemoveIfPresent(key K) bool {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	if _, ok := s.InternalMap[key]; ok {
		delete(s.InternalMap, key)
		return true
	}

	return false
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

func (s *Structure[K, V]) AtomicRemove(key K) {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	s.Remove(key)
}

func (s *Structure[K, V]) Len() int {
	s.internalLock.RLock()
	defer s.internalLock.RUnlock()

	return len(s.InternalMap)
}
