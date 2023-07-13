package hardware

import (
	"log"
	"math"
	"runtime"

	"github.com/pbnjay/memory"
	"github.com/shirou/gopsutil/cpu"
)

type HardwareUsage struct {
	CpuUsage    int32
	MemoryUsage int32
}

func GetNumberCpus() int32 {
	return int32(runtime.NumCPU())
}

func GetMemory() uint64 {
	return memory.TotalMemory()
}

func getCpuUsage() int32 {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		log.Fatal(err)
	}

	return int32(math.Ceil(percent[0]))
}

func getMemoryUsage() int32 {
	memory := float64(memory.TotalMemory()-memory.FreeMemory()) / float64(memory.TotalMemory())
	return int32(memory * 100)
}

func GetHardwareUsage() HardwareUsage {
	return HardwareUsage{
		CpuUsage:    getCpuUsage(),
		MemoryUsage: getMemoryUsage(),
	}
}
