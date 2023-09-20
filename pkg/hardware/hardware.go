package hardware

import (
	"log"
	"math"
	"runtime"

	"github.com/pbnjay/memory"
	"github.com/shirou/gopsutil/cpu"
)

type HardwareUsage struct {
	CpuUsage    uint64
	MemoryUsage uint64
}

func GetNumberCpus() uint64 {
	return uint64(runtime.NumCPU())
}

func GetMemory() uint64 {
	return memory.TotalMemory()
}

func getCpuUsage() uint64 {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		log.Fatal(err)
	}

	return uint64(math.Ceil(percent[0]))
}

func getMemoryUsage() uint64 {
	memory := float64(memory.TotalMemory()-memory.FreeMemory()) / float64(memory.TotalMemory())
	return uint64(memory * 100)
}

func GetHardwareUsage() HardwareUsage {
	return HardwareUsage{
		CpuUsage:    getCpuUsage(),
		MemoryUsage: getMemoryUsage(),
	}
}
