package workers

import (
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer/data_plane"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWorkerNodeCreation(t *testing.T) {
	mockIp := "mockIp"
	mockAPI := "mockAPI"
	mockProxy := "mockProxy"

	dataPlane := data_plane.NewDataplaneConnection(mockIp, mockAPI, mockProxy)

	assert.Equal(t, mockIp, dataPlane.GetIP())
	assert.Equal(t, mockAPI, dataPlane.GetApiPort())
	assert.Equal(t, mockProxy, dataPlane.GetProxyPort())
}
