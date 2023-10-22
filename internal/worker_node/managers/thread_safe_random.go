package managers

import (
	"math/rand"
	"sync"
	"time"
)

type ThreadSafeRandomGenerator struct {
	s *rand.Rand
	m sync.Mutex
}

func NewThreadSafeRandomGenerator() *ThreadSafeRandomGenerator {
	return &ThreadSafeRandomGenerator{
		s: rand.New(rand.NewSource(time.Now().UnixNano())),
		m: sync.Mutex{},
	}
}

func (rg *ThreadSafeRandomGenerator) Int() int {
	rg.m.Lock()
	defer rg.m.Unlock()

	return rg.s.Int()
}
