package atomic_map_counter

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

func TestNewAtomicMapCounter(t *testing.T) {
	assert.NotNil(t, NewAtomicMapCounter[int](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMapCounter[float64](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMapCounter[mockTypeStruct](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMapCounter[*mockTypeStruct](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMapCounter[interface{}](), "Atomic map should not be nil")
}

func TestNewAtomicMapCounter_AtomicGet(t *testing.T) {
	atomicMap := NewAtomicMapCounter[int]()
	assert.Zero(t, atomicMap.Get(0))
	assert.Zero(t, atomicMap.Get(1))
	assert.Zero(t, atomicMap.Get(2))
	assert.Zero(t, atomicMap.Get(3))
	assert.Zero(t, atomicMap.Get(4))
	assert.Zero(t, atomicMap.Get(5))
}

func TestNewAtomicMapCounter_AtomicDecrement(t *testing.T) {
	atomicMap := NewAtomicMapCounter[int]()
	nb := int64(10000)

	atomicMap.atomicUpdate(mockKey, nb)

	for i := nb; i >= 0; i-- {
		assert.Equal(t, i, atomicMap.Get(mockKey), "Values should be similar")
		atomicMap.AtomicDecrement(mockKey)
	}
}

func TestNewAtomicMapCounter_AtomicDelete(t *testing.T) {
	atomicMap := NewAtomicMapCounter[int]()
	nb := int64(10000)
	atomicMap.atomicUpdate(mockKey, nb)
	assert.Equal(t, nb, atomicMap.Get(mockKey), "Values should be similar")
	atomicMap.atomicUpdate(mockKey, -nb)
	assert.Equal(t, int64(0), atomicMap.Get(mockKey), "Values should be the same")
}

func TestNewAtomicMapCounter_MultipleThreads(t *testing.T) {
	atomicMap := NewAtomicMapCounter[int]()

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
	assert.Equal(t, int64(nbThreads*size), atomicMap.Get(mockKey), "Values should be similar")
}

func TestNewAtomicMapCounter_MultipleThreadsSecond(t *testing.T) {
	atomicMap := NewAtomicMapCounter[int]()

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
	assert.Equal(t, int64(0), atomicMap.Get(mockKey), "Values should be similar")
}
