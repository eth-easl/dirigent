package gpu

type Manager interface {
	NumGPUs() uint64
	UUID(index int) string
}
