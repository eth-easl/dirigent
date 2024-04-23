package atomic_map

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockTypeStruct struct {
}

func TestNewAtomicMapString(t *testing.T) {
	assert.NotNil(t, NewAtomicMap[string, string](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[string, float64](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[bool, mockTypeStruct](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[string, *mockTypeStruct](), "Atomic map should not be nil")
	assert.NotNil(t, NewAtomicMap[int, interface{}](), "Atomic map should not be nil")
}

func TestAtomicMap_AtomicGet(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	assert.False(t, atomicMap.Find(""))
	assert.False(t, atomicMap.Find(""))
	assert.False(t, atomicMap.Find(""))
	assert.False(t, atomicMap.Find(""))
	assert.False(t, atomicMap.Find(""))
	assert.False(t, atomicMap.Find(""))
}

func TestAtomicMap_CopyOnGet(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	atomicMap.Set("france", "paris")
	paris := atomicMap.GetUnsafe("france")
	atomicMap.Set("france", "paris2")
	assert.Equal(t, paris, "paris", "Values should be equal")
}

func TestAtomicMap_AtomicAdd(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	atomicMap.Set("france", "paris")
	value, ok := atomicMap.Get("france")
	assert.Truef(t, ok, "Value should be present in the map")
	assert.Equal(t, "paris", value, "Values should be similar")
}

func TestAtomicMap_AtomicDelete(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	atomicMap.Set("france", "paris")
	value, ok := atomicMap.Get("france")
	assert.True(t, ok, "Value should be present in the map")
	assert.Equal(t, "paris", value, "Values should be similar")

	atomicMap.RemoveKey("france")
	_, ok = atomicMap.Get("france")
	assert.False(t, ok, "Value should not be present in the map")
}

func TestAtomicMap_AvailableAfterDelete(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	atomicMap.Set("france", "paris")
	paris := atomicMap.GetUnsafe("france")
	atomicMap.RemoveKey("france")
	assert.Equal(t, paris, "paris", "Values should be equal")
}

func TestAtomicMap_AtomicRange(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	atomicMap.Set("france", "paris")
	atomicMap.Set("switzerland", "bern")
	atomicMap.Set("germany", "berlin")

	keys := atomicMap.Keys()
	assert.Len(t, keys, 3, "Map should have 3 keys")

	values := atomicMap.Values()
	assert.Len(t, values, 3, "Map should have 3 entries")

	keys, values = atomicMap.KeyValues()
	assert.Len(t, keys, 3, "Map should have 3 keys")
	assert.Len(t, values, 3, "Map should have 3 entries")
}

func TestAtomicMap_AtomicLenn(t *testing.T) {
	atomicMap := NewAtomicMap[string, string]()
	atomicMap.Set("france", "paris")
	atomicMap.Set("switzerland", "bern")
	atomicMap.Set("germany", "berlin")
	atomicMap.Set("italy", "roma")
	atomicMap.Set("spain", "madrid")

	assert.Equal(t, atomicMap.Len(), 5, "Map should have a length of 5")
}

func TestAtomicMap_MultipleThreads(t *testing.T) {
	atomicMap := NewAtomicMap[string, int]()

	var wg sync.WaitGroup

	size := 1000
	wg.Add(size)

	for i := 0; i < size; i++ {
		go func(idx int) {
			key := fmt.Sprintf("key:%d", idx)
			atomicMap.Set(key, 0)
			atomicMap.RemoveKey(key)
			atomicMap.Set(key, 0xbeef)
			wg.Done()
		}(i)
	}

	wg.Wait()

	keys := atomicMap.Keys()
	assert.Lenf(t, keys, size, "Map should have %d keys", size)
}
