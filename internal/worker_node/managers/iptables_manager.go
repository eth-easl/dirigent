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

package managers

import (
	"fmt"
	"os/exec"
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
		logrus.Errorf("Error adding a PREROUTING rule for %d->%s:%d - %s", sourcePort, destIP, destPort, err.Error())
	}
	logrus.Debugf("Added IP table rule for external traffic any:%d -> %s:%d", sourcePort, destIP, destPort)

	err = ipt.Append(
		"nat",
		"OUTPUT",
		"-p", "tcp", "-o", "lo", "--dport", strconv.Itoa(sourcePort), "-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", destIP, destPort),
	)
	if err != nil {
		logrus.Errorf("Error adding an OUTPUT rule for %d->%s:%d - %s", sourcePort, destIP, destPort, err.Error())
	}
	logrus.Debugf("Added IP table rule for localhost traffic any:%d -> %s:%d", sourcePort, destIP, destPort)

	err = ipt.AppendUnique(
		"nat",
		"POSTROUTING",
		"-j", "MASQUERADE",
	)
	if err != nil {
		logrus.Errorf("Error adding a POSTROUTING MASQUERADE - %s", err.Error())
	}

	err = exec.Command("sudo", "iptables", "-P", "FORWARD", "ACCEPT").Run()
	if err != nil {
		logrus.Errorf("Error changing IP routing policy FORWARD ACCEPT - %s", err.Error())
	}
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
