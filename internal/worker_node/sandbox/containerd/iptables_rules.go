package containerd

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/coreos/go-iptables/iptables"
)

func NewIptablesUtil() (*iptables.IPTables, error) {
	return iptables.New()
}

func AddRules(ipt *iptables.IPTables, sourcePort int, destIP string, destPort int) {
	err := ipt.Append(
		"nat",
		"PREROUTING",
		"-p", "tcp", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error adding a PREROUTING rule for %d->%s:%d", sourcePort, destIP, destPort)
		logrus.Error(err)
	}

	err = ipt.Append(
		"nat",
		"OUTPUT",
		"-p", "tcp", "-o", "lo", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error adding an OUTPUT rule for %d->%s:%d", sourcePort, destIP, destPort)
		logrus.Error(err)
	}

	err = ipt.AppendUnique(
		"nat",
		"POSTROUTING",
		"-j", "MASQUERADE",
	)
	if err != nil {
		logrus.Error("Error adding a POSTROUTING MASQUERADE")
		logrus.Error(err)
	}
}

func DeleteRules(ipt *iptables.IPTables, sourcePort int, destIP string, destPort int) {
	ipt.Delete(
		"nat",
		"PREROUTING",
		"-p", "tcp", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)

	ipt.Delete(
		"nat",
		"OUTPUT",
		"-p", "tcp", "-o", "lo", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)

	ipt.Delete(
		"nat",
		"POSTROUTING",
		"-j", "MASQUERADE",
	)
}
