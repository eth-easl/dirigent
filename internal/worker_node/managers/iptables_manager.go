package managers

import (
	"fmt"
	"strconv"

	"github.com/coreos/go-iptables/iptables"
	"github.com/sirupsen/logrus"
)

func NewIptablesUtil() (*iptables.IPTables, error) {
	ipt, err := iptables.New()
	if err != nil {
		logrus.Fatal("Error creating Golang iptables binding.")
	}

	err = ipt.ChangePolicy("filter", "FORWARD", "ACCEPT")
	if err != nil {
		logrus.Fatalf("Error changing iptables filter table FORWARD policy to ACCEPT - %s", err.Error())
	}
	logrus.Debugf("iptables filter FORWARD policy set to ACCEPT")

	return ipt, err
}

func AddRules(ipt *iptables.IPTables, sourcePort int, destIP string, destPort int) {
	// For cluster setup
	// E.g., map ipv4_packet(src: data_plane, dst: this_node:random_port) to ipv4_packet(src: data_plane, dst: sandbox_ip:sandbox_port)
	err := ipt.AppendUnique(
		"nat",
		"PREROUTING",
		"-p", "tcp", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error adding a PREROUTING rule for %d->%s:%d - %s", sourcePort, destIP, destPort, err.Error())
	}
	logrus.Debugf("Added IP table rule for external traffic any:%d -> %s:%d", sourcePort, destIP, destPort)

	// For local development setup
	// E.g., map ipv4_packet(src: data_plane, dst: this_node:random_port) to ipv4_packet(src: data_plane, dst: sandbox_ip:sandbox_port)
	err = ipt.AppendUnique(
		"nat",
		"OUTPUT",
		"-p", "tcp", "-o", "lo", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error adding an OUTPUT rule for %d->%s:%d - %s", sourcePort, destIP, destPort, err.Error())
	}
	logrus.Debugf("Added IP table rule for localhost traffic any:%d -> %s:%d", sourcePort, destIP, destPort)

	// This is so that the packets know the way out. Source IP:port will become the node local.
	err = ipt.AppendUnique(
		"nat",
		"POSTROUTING",
		"-j", "MASQUERADE",
	)
	if err != nil {
		logrus.Errorf("Error adding a POSTROUTING MASQUERADE - %s", err.Error())
	}
	logrus.Debugf("Added a POSTROUTING MASQUERADE if not existed.")
}

func DeleteRules(ipt *iptables.IPTables, sourcePort int, destIP string, destPort int) {
	err := ipt.Delete(
		"nat",
		"PREROUTING",
		"-p", "tcp", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error deleting a PREROUTING rule for %d->%s:%d - %s", sourcePort, destIP, destPort, err.Error())
	}

	err = ipt.Delete(
		"nat",
		"OUTPUT",
		"-p", "tcp", "-o", "lo", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error deleting an OUTPUT rule for %d->%s:%d - %s", sourcePort, destIP, destPort, err.Error())
	}

	// TODO: make sure it's deleted only after the last sandbox is deleted
	/*err = ipt.Delete(
		"nat",
		"POSTROUTING",
		"-j", "MASQUERADE",
	)
	if err != nil {
		logrus.Errorf("Error deleting a POSTROUTING MASQUERADE - %s", err.Error())
	}*/
}
