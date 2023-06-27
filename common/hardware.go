package common

import (
	"github.com/pbnjay/memory"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
	"log"
	"math"
	"runtime"
	"time"
)

type HarwareUsage struct {
	CpuUsage    int32
	MemoryUsage int32
}

func GetNumberCpus() int32 {
	return int32(runtime.NumCPU())
}

func GetMemory() int32 {
	return int32(memory.TotalMemory())
}

func getCpuUsage() int32 {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Fatal(err)
	}
	return int32(math.Ceil(percent[0]))
}

func getMemoryUsage() int32 {
	memory, err := mem.VirtualMemory()
	if err != nil {
		log.Fatal(err)
	}
	return int32(math.Ceil(memory.UsedPercent))
}

func GetHardwareUsage() HarwareUsage {
	return HarwareUsage{
		CpuUsage:    getCpuUsage(),
		MemoryUsage: getMemoryUsage(),
	}
}
