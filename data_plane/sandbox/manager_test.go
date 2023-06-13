package sandbox

import (
	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
	"testing"
)

func TestCreateSandboxWithTeardown(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	cli := GetDockerClient()

	containerConfig := &container.Config{
		Image: "docker.io/cvetkovic/empty_function:latest",
	}
	hostConfig := &container.HostConfig{}

	id, err := CreateSandbox(cli, hostConfig, containerConfig)
	if err != nil {
		t.Fatal(err)
	}

	err = DeleteSandbox(cli, id)
	if err != nil {
		t.Error(err)
	}
}
