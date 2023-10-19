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
	return &ProcessMonitor{}
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
