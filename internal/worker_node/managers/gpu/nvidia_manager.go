//go:build nvidia_gpu

package gpu

import (
	"errors"
	"fmt"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/sirupsen/logrus"
)

type NvidiaGPUManager struct {
	Manager

	uuids []string
}

func newNvidiaGPUManager() *NvidiaGPUManager {
	uuids, err := getUUIDs()
	if err != nil {
		logrus.Errorf("failed to get GPU UUIDs: %v", err)
	}

	logrus.Debugf("Found %d GPUs", len(uuids))
	return &NvidiaGPUManager{uuids: uuids}
}

func getUUIDs() ([]string, error) {
	if ret := nvml.Init(); !errors.Is(ret, nvml.SUCCESS) {
		return []string{}, fmt.Errorf("failed to initialize NVML: %v", nvml.ErrorString(ret))
	}
	defer func() {
		if ret := nvml.Shutdown(); !errors.Is(ret, nvml.SUCCESS) {
			logrus.Errorf("failed to shutdown NVML: %v", nvml.ErrorString(ret))
		}
	}()

	count, ret := nvml.DeviceGetCount()
	if !errors.Is(ret, nvml.SUCCESS) {
		return []string{}, fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}

	uuids := make([]string, count)
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if !errors.Is(ret, nvml.SUCCESS) {
			return []string{}, fmt.Errorf("failed to get device at index %d: %v", i, nvml.ErrorString(ret))
		}

		uuid, ret := device.GetUUID()
		if !errors.Is(ret, nvml.SUCCESS) {
			return []string{}, fmt.Errorf("failed to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
		}

		uuids[i] = uuid
	}

	return uuids, nil
}

func (m NvidiaGPUManager) NumGPUs() uint64 {
	return uint64(len(m.uuids))
}

func (m NvidiaGPUManager) UUID(index int) string {
	return m.uuids[index]
}
