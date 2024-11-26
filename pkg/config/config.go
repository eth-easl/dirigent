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

package config

import (
	"cluster_manager/pkg/network"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type ControlPlaneConfig struct {
	Port                       string         `mapstructure:"port"`
	Replicas                   []string       `mapstructure:"replicas"`
	RegistrationServer         string         `mapstructure:"registrationServer"`
	RegistrationServerReplicas []string       `mapstructure:"registrationServerReplicas"`
	Verbosity                  string         `mapstructure:"verbosity"`
	TraceOutputFolder          string         `mapstructure:"traceOutputFolder"`
	PlacementPolicy            string         `mapstructure:"placementPolicy"`
	Persistence                bool           `mapstructure:"persistence"`
	Profiler                   ProfilerConfig `mapstructure:"profiler"`
	RedisConf                  RedisConf      `mapstructure:"redis"`
	Reconstruct                bool           `mapstructure:"reconstruct"`
	LoadBalancerAddress        string         `mapstructure:"loadBalancerAddress"`
	RemoveWorkerNode           bool           `mapstructure:"removeWorkerNode"`
	RemoveDataplane            bool           `mapstructure:"removeDataplane"`
	PrecreateSnapshots         bool           `mapstructure:"precreateSnapshots"`
	EndpointPersistence        bool           `mapstructure:"endpointPersistence"`
}

type DataPlaneConfig struct {
	DataPlaneIp         string    `mapstructure:"dataPlaneIp"`
	ControlPlaneAddress []string  `mapstructure:"controlPlaneAddress"`
	PortProxy           string    `mapstructure:"portProxy"`
	PortProxyRead       string    `mapstructure:"portProxyRead"`
	PortGRPC            string    `mapstructure:"portGRPC"`
	Verbosity           string    `mapstructure:"verbosity"`
	TraceOutputFolder   string    `mapstructure:"traceOutputFolder"`
	LoadBalancingPolicy string    `mapstructure:"loadBalancingPolicy"`
	Async               bool      `mapstructure:"async"`
	RedisConf           RedisConf `mapstructure:"redis"`
	PersistRequests     bool      `mapstructure:"persistRequests"`
	NumberRetries       int       `mapstructure:"numberRetries"`
}

type WorkerNodeConfig struct {
	WorkerNodeIP        string   `mapstructure:"workerNodeIp"`
	ControlPlaneAddress []string `mapstructure:"controlPlaneAddress"`
	Port                int      `mapstructure:"port"`
	Verbosity           string   `mapstructure:"verbosity"`
	CRIType             string   `mapstructure:"criType"`
	CRIPath             string   `mapstructure:"criPath"`
	CNIConfigPath       string   `mapstructure:"cniConfigPath"`
	PrefetchImage       bool     `mapstructure:"prefetchImage"`

	FirecrackerKernel           string `mapstructure:"firecrackerKernel"`
	FirecrackerFileSystem       string `mapstructure:"firecrackerFileSystem"`
	FirecrackerInternalIPPrefix string `mapstructure:"firecrackerInternalIPPrefix"`
	FirecrackerExposedIPPrefix  string `mapstructure:"firecrackerExposedIPPrefix"`
	FirecrackerVMDebugMode      bool   `mapstructure:"firecrackerVMDebugMode"`
	FirecrackerUseSnapshots     bool   `mapstructure:"firecrackerUseSnapshots"`
	FirecrackerNetworkPoolSize  int    `mapstructure:"firecrackerNetworkPoolSize"`
}

type ProfilerConfig struct {
	Enable bool `mapstructure:"enable"`
	Mutex  bool `mapstructure:"mutex"`
}

type RedisConf struct {
	Address         string   `mapstructure:"address"`
	DockerAddress   string   `mapstructure:"dockerAddress"`
	Password        string   `mapstructure:"password"`
	Replicas        []string `mapstructure:"replicas"`
	Db              int      `mapstructure:"db"`
	FullPersistence bool     `mapstructure:"fullPersistence"`
}

func parseConfigPath(configPath string) (string, string, string) {
	configFolder, configName := filepath.Split(configPath)
	configName = strings.TrimSuffix(configName, filepath.Ext(configName))
	configType := strings.ReplaceAll(filepath.Ext(configPath), ".", "")

	if configFolder == "" {
		configFolder = "./"
	}

	return configFolder, configName, configType
}

func setupViper(configPath string) error {
	configFolder, configName, configType := parseConfigPath(configPath)

	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configFolder)
	viper.AutomaticEnv()

	return viper.ReadInConfig()
}

func ReadControlPlaneConfiguration(configPath string) (ControlPlaneConfig, error) {
	err := setupViper(configPath)
	if err != nil {
		return ControlPlaneConfig{}, err
	}

	controlPlaneConfig := ControlPlaneConfig{}

	err = viper.Unmarshal(&controlPlaneConfig)
	if err != nil {
		return ControlPlaneConfig{}, err
	}

	if len(controlPlaneConfig.Replicas)%2 == 1 {
		logrus.Fatal("There must be odd number of control plane replicas participating in the leader election.")
	}

	return controlPlaneConfig, nil
}

func ReadDataPlaneConfiguration(configPath string) (DataPlaneConfig, error) {
	err := setupViper(configPath)
	if err != nil {
		return DataPlaneConfig{}, err
	}

	dataPlaneConfig := DataPlaneConfig{}

	err = viper.Unmarshal(&dataPlaneConfig)
	if err != nil {
		return DataPlaneConfig{}, err
	}

	if dataPlaneConfig.DataPlaneIp == "dynamic" {
		dataPlaneConfig.DataPlaneIp = network.GetLocalIP()
	}

	return dataPlaneConfig, nil
}

func ReadWorkedNodeConfiguration(configPath string) (WorkerNodeConfig, error) {
	err := setupViper(configPath)
	if err != nil {
		return WorkerNodeConfig{}, err
	}

	workerNodeConfig := WorkerNodeConfig{}

	err = viper.Unmarshal(&workerNodeConfig)
	if err != nil {
		return WorkerNodeConfig{}, err
	}

	return workerNodeConfig, nil
}
