package control_plane

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkerNodeCreation(t *testing.T) {
	mockIp := "mockIp"
	mockAPI := "mockAPI"
	mockProxy := "mockProxy"

	dataPlane := NewDataplaneConnection(mockIp, mockAPI, mockProxy)

	assert.Equal(t, mockIp, dataPlane.GetIP())
	assert.Equal(t, mockAPI, dataPlane.GetApiPort())
	assert.Equal(t, mockProxy, dataPlane.GetProxyPort())
}
