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
	"github.com/fearful-symmetry/garlic"
	"github.com/sirupsen/logrus"
	"sync"
)

const (
	PCNStartupTries = 10
)

type ProcessMonitor struct {
	PCN garlic.CnConn

	// NotifyChannels PID -> exit code channel
	NotifyChannels map[uint32]chan uint32
	sync.RWMutex
}

func NewProcessMonitor() *ProcessMonitor {
	var cn garlic.CnConn
	var err error

	for i := 1; i <= PCNStartupTries; i++ {
		cn, err = garlic.DialPCNWithEvents([]garlic.EventType{garlic.ProcEventExit})
		if err != nil {
			logrus.Errorf("Failed do start process monitor (attempt %d): (error: %s)", i, err.Error())
			continue
		}

		break
	}

	if err != nil {
		// kill worker node daemon if process monitor doesn't start
		logrus.Fatalf("Failed do start process monitor. Terminating worker daemon.")
	}

	pm := &ProcessMonitor{
		PCN:            cn,
		NotifyChannels: make(map[uint32]chan uint32),
		RWMutex:        sync.RWMutex{},
	}

	go pm.internalEventHandler()

	return pm
}

func (pm *ProcessMonitor) AddChannel(PID uint32, notifyChannel chan uint32) {
	pm.Lock()
	defer pm.Unlock()

	pm.NotifyChannels[PID] = notifyChannel
}

func (pm *ProcessMonitor) RemoveChannel(PID uint32) {
	pm.Lock()
	defer pm.Unlock()

	delete(pm.NotifyChannels, PID)
}

func (pm *ProcessMonitor) internalEventHandler() {
	for {
		events, err := pm.PCN.ReadPCN()
		if err != nil {
			logrus.Errorf("Error while monitoring processes. Exiting monitoring loop.")
			break
		}

		if len(events) == 0 {
			continue
		}

		pm.RLock()
		for _, event := range events {
			eData := event.EventData
			PID := eData.Pid()

			ch, ok := pm.NotifyChannels[PID]
			if ok {
				ch <- extractExitCode(&eData)
			}
		}
		pm.RUnlock()
	}
}

func extractExitCode(data *garlic.EventData) uint32 {
	switch (*data).(type) {
	case garlic.Exit:
		// Value of 128 -- https://tldp.org/LDP/abs/html/exitcodes.html
		return (*data).(garlic.Exit).ExitCode + 128
	default:
		logrus.Error("Garlic return wrong type of event.")
	}

	return 255
}
