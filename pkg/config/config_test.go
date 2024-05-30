package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	readConfigError  string = "Failed to read configuration"
	parseConfigError string = "Failed to parse configuration"
)

func TestReadControlPlaneConfiguration(t *testing.T) {
	config, err := ReadControlPlaneConfiguration("../../cmd/master_node/config.yaml")
	assert.NoError(t, err, readConfigError)

	assert.True(t, len(config.Verbosity) > 0, parseConfigError)
}

func TestReadDataPlaneConfiguration(t *testing.T) {
	config, err := ReadDataPlaneConfiguration("../../cmd/data_plane/config.yaml")
	assert.NoError(t, err, readConfigError)

	assert.True(t, len(config.Verbosity) > 0, parseConfigError)
}

func TestReadWorkedNodeConfiguration(t *testing.T) {
	config, err := ReadWorkedNodeConfiguration("../../cmd/worker_node/config.yaml")
	assert.NoError(t, err, readConfigError)

	assert.True(t, len(config.Verbosity) > 0, parseConfigError)
}

func TestGetReaderFromPath(t *testing.T) {
	type want struct {
		configFolder string
		configName   string
		configType   string
	}

	tests := []struct {
		name     string
		input    string
		expected want
	}{
		{
			name:  "Simple smoke input",
			input: "test/test.yaml",
			expected: want{
				configFolder: "test/",
				configName:   "test",
				configType:   "yaml",
			},
		},
		{
			name:  "Test with no folder",
			input: "filename.txt",
			expected: want{
				configFolder: "./",
				configName:   "filename",
				configType:   "txt",
			},
		},
		{
			name:  "Test with german chars (test with umlauts)",
			input: "ü/ä.yaml",
			expected: want{
				configFolder: "ü/",
				configName:   "ä",
				configType:   "yaml",
			},
		},
		{
			name:  "Test with french chars (test with accents)",
			input: "hôte/équipe.yaml",
			expected: want{
				configFolder: "hôte/",
				configName:   "équipe",
				configType:   "yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFolder, configName, configType := parseConfigPath(tt.input)

			assert.Equal(t, configFolder, tt.expected.configFolder, "Not the correct config.yaml folder.")
			assert.Equal(t, configName, tt.expected.configName, "Not the correct config.yaml name.")
			assert.Equal(t, configType, tt.expected.configType, "Not the correct config.yaml type. ")
		})
	}
}
