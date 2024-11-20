package managers

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/sirupsen/logrus"
)

type GPUManager struct {
	uuids []string
}

func getUUIDs() ([]string, error) {
	if ret := nvml.Init(); ret != nvml.SUCCESS {
		return []string{}, fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer func() {
		if ret := nvml.Shutdown(); ret != nvml.SUCCESS {
			logrus.Errorf("failed to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()
	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return []string{}, fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}
	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			return []string{}, fmt.Errorf("failed to get device at index %d: %v", i, nvml.ErrorString(ret))
		}
		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			return []string{}, fmt.Errorf("failed to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
		}
		uuids[i] = uuid
	}
	return uuids, nil
}

func NewGPUManager() *GPUManager {
	uuids, err := getUUIDs()
	if err != nil {
		logrus.Errorf("failed to get GPU UUIDs: %v", err)
	}
	logrus.Debugf("Found %d GPUs", len(uuids))
	return &GPUManager{uuids}
}

func (m *GPUManager) NumGPUs() uint64 {
	return uint64(len(m.uuids))
}

func (m *GPUManager) UUID(index int) string {
	return m.uuids[index]
}
