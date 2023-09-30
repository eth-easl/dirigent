package synchronization

import "sync"

type SyncStructure[K comparable, V any] interface {
	SetIfAbsent(key K, value V) bool // Atomic operation
	RemoveIfPresent(key K) bool      // Atomic operation

	Lock()
	Unlock()

	Set(key K, value V) // Not thread safe
	Get(key K) V        // Not thread safe

	AtomicSet(key K, value V)
	AtomicGet(key K) V

	Len() int // Atomic operation
}

func NewControlPlaneSyncStructure[K comparable, V any]() *structure[K, V] {
	return &structure[K, V]{
		internalMap:  make(map[K]V),
		internalLock: sync.RWMutex{},
	}
}

type structure[K comparable, V any] struct {
	internalMap  map[K]V
	internalLock sync.RWMutex
}

func (s *structure[K, V]) SetIfAbsent(key K, value V) bool {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	if _, ok := s.internalMap[key]; !ok {
		s.internalMap[key] = value
		return true
	}

	return false
}

func (s *structure[K, V]) RemoveIfPresent(key K) bool {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	if _, ok := s.internalMap[key]; ok {
		delete(s.internalMap, key)
		return true
	}

	return false
}

func (s *structure[K, V]) Lock() {
	s.internalLock.Lock()
}

func (s *structure[K, V]) Unlock() {
	s.internalLock.Unlock()
}

func (s *structure[K, V]) Set(key K, value V) {
	s.internalMap[key] = value
}

func (s *structure[K, V]) Get(key K) V {
	return s.internalMap[key]
}

func (s *structure[K, V]) AtomicSet(key K, value V) {
	s.internalLock.Lock()
	defer s.internalLock.Unlock()

	s.Set(key, value)
}

func (s *structure[K, V]) AtomicGet(key K) V {
	s.internalLock.RLock()
	defer s.internalLock.RUnlock()

	return s.Get(key)
}

func (s *structure[K, V]) Len() int {
	s.internalLock.RLock()
	defer s.internalLock.RUnlock()

	return len(s.internalMap)
}
