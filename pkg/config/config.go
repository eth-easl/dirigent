package config

import (
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type ControlPlaneConfig struct {
	Port               string         `mapstructure:"port"`
	PortRegistration   string         `mapstructure:"portRegistration"`
	Verbosity          string         `mapstructure:"verbosity"`
	TraceOutputFolder  string         `mapstructure:"traceOutputFolder"`
	PlacementPolicy    string         `mapstructure:"placementPolicy"`
	Persistence        bool           `mapstructure:"persistence"`
	Profiler           ProfilerConfig `mapstructure:"profiler"`
	RedisConf          RedisConf      `mapstructure:"redis"`
	Reconstruct        bool           `mapstructure:"reconstruct"`
	RemoveWorkerNode   bool           `mapstructure:"removeWorkerNode"`
	RemoveDataplane    bool           `mapstructure:"removeDataplane"`
	PrecreateSnapshots bool           `mapstructure:"precreateSnapshots"`
}

type DataPlaneConfig struct {
	DataPlaneIp         string    `mapstructure:"dataPlaneIp"`
	ControlPlaneIp      string    `mapstructure:"controlPlaneIp"`
	ControlPlanePort    string    `mapstructure:"controlPlanePort"`
	PortProxy           string    `mapstructure:"portProxy"`
	PortProxyRead       string    `mapstructure:"portProxyRead"`
	PortGRPC            string    `mapstructure:"portGRPC"`
	Verbosity           string    `mapstructure:"verbosity"`
	TraceOutputFolder   string    `mapstructure:"traceOutputFolder"`
	LoadBalancingPolicy string    `mapstructure:"loadBalancingPolicy"`
	Async               bool      `mapstructure:"async"`
	RedisConf           RedisConf `mapstructure:"redis"`
}

type WorkerNodeConfig struct {
	WorkerNodeIP     string `mapstructure:"workerNodeIp"`
	ControlPlaneIp   string `mapstructure:"controlPlaneIp"`
	ControlPlanePort string `mapstructure:"controlPlanePort"`
	Port             int    `mapstructure:"port"`
	Verbosity        string `mapstructure:"verbosity"`
	CRIType          string `mapstructure:"criType"`
	CRIPath          string `mapstructure:"criPath"`
	CNIConfigPath    string `mapstructure:"cniConfigPath"`
	PrefetchImage    bool   `mapstructure:"prefetchImage"`

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
	Address         string `mapstructure:"address"`
	Password        string `mapstructure:"password"`
	Db              int    `mapstructure:"db"`
	FullPersistence bool   `mapstructure:"fullPersistence"`
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
