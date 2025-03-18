//go:build !nvidia_gpu

package gpu

func GetGPUManager() Manager {
	return newNoGPUManager()
}
