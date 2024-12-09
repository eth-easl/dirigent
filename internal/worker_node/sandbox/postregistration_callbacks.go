package sandbox

type PostRegistrationCallback func(runtime RuntimeInterface, cidr string)

func EmptyPostRegistrationCallback(RuntimeInterface, string) {}

func ContainerdPostRegistrationCallback(runtime RuntimeInterface, cidr string) {
	runtime.ConfigureNetwork(cidr)
}
