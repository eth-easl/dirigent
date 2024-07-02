package hardware

import (
	"log"
	"runtime"

	"github.com/pbnjay/memory"
	"github.com/shirou/gopsutil/cpu"
)

type HardwareUsage struct {
	CpuUsage    float64
	MemoryUsage float64
}

func GetNumberCpus() uint64 {
	return uint64(runtime.NumCPU())
}

func GetMemory() uint64 {
	return memory.TotalMemory()
}

func getCpuUsage() float64 {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		log.Fatal(err)
	}

	return percent[0] / 100.0
}

func getMemoryUsage() float64 {
	return float64(memory.TotalMemory()-memory.FreeMemory()) / float64(memory.TotalMemory())
}

func GetHardwareUsage() HardwareUsage {
	return HardwareUsage{
		CpuUsage:    getCpuUsage(),
		MemoryUsage: getMemoryUsage(),
	}
}
