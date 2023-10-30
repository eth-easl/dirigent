package managers

import "sync/atomic"

type VETHNameGenerator struct {
	counter int32
}

func NewVETHNameGenerator() *VETHNameGenerator {
	return &VETHNameGenerator{}
}

func (gen *VETHNameGenerator) Generate() int32 {
	swapped := false
	var oldValue int32 = -1
	for !swapped {
		oldValue = atomic.LoadInt32(&gen.counter)
		newValue := oldValue + 2

		swapped = atomic.CompareAndSwapInt32(&gen.counter, oldValue, newValue)
	}

	return oldValue
}
