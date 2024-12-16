package ipam

import (
	"container/list"
	"fmt"
	"sync"
)

// MaximumClusterSize Maximum number of CIDRs - [11.0.X.X/16, 99.255.X.X/16]
const MaximumClusterSize = 89 * 256
const Offset = 11

type DynamicCIDRManager struct {
	CIDRManager

	generator int
	inUse     int

	pool  *list.List
	mutex sync.Mutex
}

func NewDynamicCIDRManager() *DynamicCIDRManager {
	m := &DynamicCIDRManager{
		pool:  &list.List{},
		mutex: sync.Mutex{},
	}
	m.populatePool()

	return m
}

func counterToFirstByte(cnt int) int {
	return Offset + int(cnt/256)
}

func counterToSecondByte(cnt int) int {
	return cnt % 256
}

// populatePool assumes that mutex has been acquired by the caller
func (m *DynamicCIDRManager) populatePool() {
	if m.pool.Len() >= 16 {
		return
	}

	for i := 0; i < 16; i++ {
		cnt := m.generator
		m.generator++

		m.pool.PushBack(fmt.Sprintf("%d.%d.0.0/16", counterToFirstByte(cnt), counterToSecondByte(cnt)))
	}
}

func (m *DynamicCIDRManager) ReserveCIDR() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.inUse >= MaximumClusterSize {
		return "", fmt.Errorf("maximum cluster size exceeded")
	}

	front := m.pool.Front()
	m.pool.Remove(front)

	if m.pool.Len() == 0 {
		m.populatePool()
	}

	m.inUse++
	return front.Value.(string), nil
}

func (m *DynamicCIDRManager) ReleaseCIDR(cidr string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.pool.PushFront(cidr)
	m.inUse--
}

func (m *DynamicCIDRManager) IsStatic() bool {
	return false
}
