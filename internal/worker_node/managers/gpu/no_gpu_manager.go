package gpu

type NoGPUManager struct {
	Manager
}

func newNoGPUManager() *NoGPUManager {
	return &NoGPUManager{}
}

func (m NoGPUManager) NumGPUs() uint64 {
	return 0
}

func (m NoGPUManager) UUID(index int) string {
	panic("UUID of 'NoGPUManager' should not have been called.")
}
