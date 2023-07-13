package config

import (
	"cluster_manager/internal/control_plane"
	"cluster_manager/internal/proxy/load_balancing"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type ControlPlaneConfig struct {
	Port              string     `mapstructure:"port"`
	PortRegistration  string     `mapstructure:"portRegistration"`
	Verbosity         string     `mapstructure:"verbosity"`
	TraceOutputFolder string     `mapstructure:"traceOutputFolder"`
	PlacementPolicy   string     `mapstructure:"placementPolicy"`
	RedisLogin        RedisLogin `mapstructure:"redis"`
}

type DataPlaneConfig struct {
	ControlPlaneIp      string `mapstructure:"controlPlaneIp"`
	ControlPlanePort    string `mapstructure:"controlPlanePort"`
	PortProxy           string `mapstructure:"portProxy"`
	PortGRPC            string `mapstructure:"portGRPC"`
	Verbosity           string `mapstructure:"verbosity"`
	TraceOutputFolder   string `mapstructure:"traceOutputFolder"`
	LoadBalancingPolicy string `mapstructure:"loadBalancingPolicy"`
}

type WorkerNodeConfig struct {
	ControlPlaneIp   string `mapstructure:"controlPlaneIp"`
	ControlPlanePort string `mapstructure:"controlPlanePort"`
	Port             int    `mapstructure:"port"`
	Verbosity        string `mapstructure:"verbosity"`
	CRIPath          string `mapstructure:"criPath"`
	CNIConfigPath    string `mapstructure:"cniConfigPath"`
}

type RedisLogin struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	Db       int    `mapstructure:"db"`
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

func ParsePlacementPolicy(controlPlaneConfig ControlPlaneConfig) control_plane.PlacementPolicy {
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		return control_plane.PLACEMENT_RANDOM
	case "round-robin":
		return control_plane.PLACEMENT_ROUND_ROBIN
	case "kubernetes":
		return control_plane.PLACEMENT_KUBERNETES
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		return control_plane.PLACEMENT_RANDOM
	}
}

func ParseLoadBalancingPolicy(dataPlaneConfig DataPlaneConfig) load_balancing.LoadBalancingPolicy {
	switch dataPlaneConfig.LoadBalancingPolicy {
	case "random":
		return load_balancing.LOAD_BALANCING_RANDOM
	case "round-robin":
		return load_balancing.LOAD_BALANCING_ROUND_ROBIN
	case "least-processed":
		return load_balancing.LOAD_BALANCING_LEAST_PROCESSED
	case "knative":
		return load_balancing.LOAD_BALANCING_KNATIVE
	default:
		logrus.Error("Failed to parse policy, default policy is random")
		return load_balancing.LOAD_BALANCING_RANDOM
	}
}
