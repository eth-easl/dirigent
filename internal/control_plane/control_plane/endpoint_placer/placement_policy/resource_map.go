package placement_policy

import "github.com/sirupsen/logrus"

const (
	RM_CPU_KEY    = "cpu"
	RM_MEMORY_KEY = "memory"
)

type ResourceMap struct {
	resources map[string]uint64
	image     string
}

func (r *ResourceMap) GetCpu() uint64 {
	return r.GetByKey(RM_CPU_KEY)
}

func (r *ResourceMap) GetMemory() uint64 {
	return r.GetByKey(RM_MEMORY_KEY)
}

func (r *ResourceMap) GetImage() string {
	return r.image
}

func (r *ResourceMap) SetCpu(v uint64) {
	r.resources[RM_CPU_KEY] = v
}

func (r *ResourceMap) SetMemory(v uint64) {
	r.resources[RM_MEMORY_KEY] = v
}

func (r *ResourceMap) SetImage(image string) {
	r.image = image
}

func (r *ResourceMap) ResourceKeys() []string {
	keys := make([]string, 0)

	for k := range r.resources {
		keys = append(keys, k)
	}

	return keys
}

func (r *ResourceMap) GetByKey(key string) uint64 {
	if v, ok := r.resources[key]; ok {
		return v
	} else {
		logrus.Fatal("There is no resource with the given key.")
		return 0
	}
}

func (r *ResourceMap) SumAllResourceTypes() uint64 {
	var sum uint64 = 0

	for _, key := range r.ResourceKeys() {
		req := r.GetByKey(key)
		sum += req
	}

	return sum
}

// CreateResourceMap CPU in milliCPUs.
func CreateResourceMap(cpu, memory uint64, image string) *ResourceMap {
	r := &ResourceMap{
		resources: make(map[string]uint64),
		image:     image,
	}

	r.SetCpu(cpu)
	r.SetMemory(memory)

	return r
}

func ExtendCPU(resourceMap *ResourceMap) *ResourceMap {
	resourceMap.SetCpu(resourceMap.GetCpu() * 1000)

	return resourceMap
}

func SumResources(a, b *ResourceMap) *ResourceMap {
	return CreateResourceMap(
		a.GetCpu()+b.GetCpu(),
		a.GetMemory()+b.GetMemory(),
		a.GetImage(),
	)
}

func SubtractResources(a, b *ResourceMap) *ResourceMap {
	return CreateResourceMap(
		a.GetCpu()-b.GetCpu(),
		a.GetMemory()-b.GetMemory(),
		a.GetImage(),
	)
}
