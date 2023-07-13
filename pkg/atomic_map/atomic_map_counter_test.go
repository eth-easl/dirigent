package atomic_map

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	mockKey int = iota
)

type mockTypeStruct struct {
	mockField  int
	mockField2 string
}

func TestNewAtomicMap(t *testing.T) {
	assert.NotNil(t, NewAtomicMap[int](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[float64](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[mockTypeStruct](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[*mockTypeStruct](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[interface{}](), "Atomic map should not be nil")
}

func TestAtomicMap_AtomicGet(t *testing.T) {
	atomicMap := NewAtomicMap[int]()
	assert.Zero(t, atomicMap.AtomicGet(0))
	assert.Zero(t, atomicMap.AtomicGet(1))
	assert.Zero(t, atomicMap.AtomicGet(2))
	assert.Zero(t, atomicMap.AtomicGet(3))
	assert.Zero(t, atomicMap.AtomicGet(4))
	assert.Zero(t, atomicMap.AtomicGet(5))
}

func TestAtomicMap_AtomicDecrement(t *testing.T) {
	atomicMap := NewAtomicMap[int]()
	nb := int64(10000)

	atomicMap.AtomicAdd(mockKey, nb)

	for i := nb; i >= 0; i-- {
		assert.Equal(t, i, atomicMap.AtomicGet(mockKey), "Values should be similar")
		atomicMap.AtomicDecrement(mockKey)
	}
}

func TestAtomicMap_AtomicDelete(t *testing.T) {
	atomicMap := NewAtomicMap[int]()
	nb := int64(10000)
	atomicMap.AtomicAdd(mockKey, nb)
	assert.Equal(t, nb, atomicMap.AtomicGet(mockKey), "Values should be similar")
	atomicMap.AtomicAdd(mockKey, -nb)
	assert.Equal(t, int64(0), atomicMap.AtomicGet(mockKey), "Values should be the same")
}

func TestAtomicMap_MultipleThreads(t *testing.T) {
	atomicMap := NewAtomicMap[int]()

	var wg sync.WaitGroup

	size := 1000
	nbThreads := 1000

	wg.Add(nbThreads)

	for i := 0; i < nbThreads; i++ {
		go func() {
			for j := 0; j < size; j++ {
				atomicMap.AtomicIncrement(mockKey)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(nbThreads*size), atomicMap.AtomicGet(mockKey), "Values should be similar")
}

func TestAtomicMap_MultipleThreadsSecond(t *testing.T) {
	atomicMap := NewAtomicMap[int]()

	var wg sync.WaitGroup

	size := 1000
	nbThreads := 1000

	wg.Add(nbThreads)

	for i := 0; i < nbThreads; i++ {
		go func() {
			for j := 0; j < size; j++ {
				atomicMap.AtomicIncrement(mockKey)
				atomicMap.AtomicDecrement(mockKey)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	assert.Equal(t, int64(0), atomicMap.AtomicGet(mockKey), "Values should be similar")
}
