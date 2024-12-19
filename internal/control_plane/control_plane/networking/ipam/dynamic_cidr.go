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

	disablePoolPopulation bool
}

func NewDynamicCIDRManager() *DynamicCIDRManager {
	m := &DynamicCIDRManager{
		pool:  &list.List{},
		mutex: sync.Mutex{},
	}

	return m
}

func counterToFirstByte(cnt int) int {
	return Offset + int(cnt/256)
}

func counterToSecondByte(cnt int) int {
	return cnt % 256
}

func cidrToCounter(cidr string) int {
	var s1, s2, s3, s4, s5 int
	_, err := fmt.Sscanf(cidr, "%d.%d.%d.%d/%d", &s1, &s2, &s3, &s4, &s5)
	if err != nil {
		return -1
	}

	return (s1-Offset)*256 + s2
}

func counterToString(cnt int) string {
	return fmt.Sprintf("%d.%d.0.0/16", counterToFirstByte(cnt), counterToSecondByte(cnt))
}

// populatePool assumes that mutex has been acquired by the caller
func (m *DynamicCIDRManager) populatePool() {
	const maxPoolSize = 16
	if m.pool.Len() >= maxPoolSize {
		return
	}

	for m.pool.Len() < maxPoolSize {
		m.generateOnce(true)
	}
}

// generateOnce assumes that mutex has been acquired by the caller
func (m *DynamicCIDRManager) generateOnce(toAdd bool) {
	cnt := m.generator
	m.generator++

	if toAdd {
		m.pool.PushBack(counterToString(cnt))
	}
}

func (m *DynamicCIDRManager) ReserveCIDRs(cidrs []string) {
	if len(cidrs) == 0 {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	hm := make(map[string]bool)

	// find maximum counter
	maximum := cidrToCounter(cidrs[0])
	hm[cidrs[0]] = true

	for i := 1; i < len(cidrs); i++ {
		hm[cidrs[i]] = true

		current := cidrToCounter(cidrs[i])
		if maximum < current {
			maximum = current
		}
	}

	// allocate until maximum count achieved
	for i := 0; i <= maximum; i++ {
		_, exists := hm[counterToString(i)]
		m.generateOnce(!exists)
	}
}

func (m *DynamicCIDRManager) GetUnallocatedCIDR() (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.inUse >= MaximumClusterSize {
		return "", fmt.Errorf("maximum cluster size exceeded")
	}

	if m.pool.Len() == 0 && !m.disablePoolPopulation {
		m.populatePool()
	}

	front := m.pool.Front()
	m.pool.Remove(front)

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

func (m *DynamicCIDRManager) disableFurtherPopulation() {
	m.disablePoolPopulation = true
}

func (m *DynamicCIDRManager) poolLength() int {
	return m.pool.Len()
}
