/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
