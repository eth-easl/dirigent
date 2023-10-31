package network

import (
	"github.com/sirupsen/logrus"
	"os/exec"
)

func GetLocalIP() string {
	cmd := exec.Command("bash", "-c", "/usr/bin/netstat -ie | grep -B1 '10.0.1' | sed -n 2p | tr -s ' ' | cut -d ' ' -f 3")
	stdout, err := cmd.Output()

	if err != nil {
		logrus.Fatal(err)
	}

	ip := string(stdout)
	ip = ip[0 : len(ip)-1]

	return ip
}
