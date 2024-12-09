package config

import (
	"cluster_manager/pkg/network"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

type ControlPlaneConfig struct {
	Port                       string        `mapstructure:"port"`
	Replicas                   []string      `mapstructure:"replicas"`
	RegistrationServer         string        `mapstructure:"registrationServer"`
	RegistrationServerReplicas []string      `mapstructure:"registrationServerReplicas"`
	Verbosity                  string        `mapstructure:"verbosity"`
	TraceOutputFolder          string        `mapstructure:"traceOutputFolder"`
	PlacementPolicy            string        `mapstructure:"placementPolicy"`
	Persistence                bool          `mapstructure:"persistence"`
	RedisConf                  RedisConf     `mapstructure:"redis"`
	Reconstruct                bool          `mapstructure:"reconstruct"`
	LoadBalancerAddress        string        `mapstructure:"loadBalancerAddress"`
	RemoveWorkerNode           bool          `mapstructure:"removeWorkerNode"`
	RemoveDataplane            bool          `mapstructure:"removeDataplane"`
	PrecreateSnapshots         bool          `mapstructure:"precreateSnapshots"`
	Autoscaler                 string        `mapstructure:"autoscaler"`
	AutoscalingPeriod          time.Duration `mapstructure:"autoscalingPeriod"`
	DefaultWFPartitionMethod   string        `mapstructure:"defaultWFPartitionMethod"`
	ImageStorage               string        `mapstructure:"imageStorage"`
}

type DataPlaneConfig struct {
	DataPlaneIp                         string    `mapstructure:"dataPlaneIp"`
	ControlPlaneAddress                 []string  `mapstructure:"controlPlaneAddress"`
	PortProxy                           string    `mapstructure:"portProxy"`
	PortProxyRead                       string    `mapstructure:"portProxyRead"`
	PortGRPC                            string    `mapstructure:"portGRPC"`
	Verbosity                           string    `mapstructure:"verbosity"`
	TraceOutputFolder                   string    `mapstructure:"traceOutputFolder"`
	LoadBalancingPolicy                 string    `mapstructure:"loadBalancingPolicy"`
	Async                               AsyncConf `mapstructure:"async"`
	RedisConf                           RedisConf `mapstructure:"redis"`
	ControlPlaneNotifyIntervalInMinutes int       `mapstructure:"controlPlaneNotifyIntervalMinutes"`
	WorkflowDefaultScheduler            string    `mapstructure:"workflowDefaultScheduler"`
	WorkflowPreferredWorkerParallelism  int       `mapstructure:"workflowPreferredWorkerParallelism"`
}

type WorkerNodeConfig struct {
	WorkerNodeIP        string            `mapstructure:"workerNodeIp"`
	ControlPlaneAddress []string          `mapstructure:"controlPlaneAddress"`
	Port                int               `mapstructure:"port"`
	Verbosity           string            `mapstructure:"verbosity"`
	CRIType             string            `mapstructure:"criType"`
	CPUConstraints      bool              `mapstructure:"cpuConstraints"`
	Containerd          ContainerdConfig  `mapstructure:"containerd"`
	Firecracker         FirecrackerConfig `mapstructure:"firecracker"`
	Dandelion           DandelionConfig   `mapstructure:"dandelion"`
}

type ContainerdConfig struct {
	CRIPath       string `mapstructure:"criPath"`
	CNIConfigPath string `mapstructure:"cniConfigPath"`
	PrefetchImage bool   `mapstructure:"prefetchImage"`
}

type FirecrackerConfig struct {
	Kernel           string `mapstructure:"Kernel"`
	FileSystem       string `mapstructure:"FileSystem"`
	InternalIPPrefix string `mapstructure:"InternalIPPrefix"`
	ExposedIPPrefix  string `mapstructure:"ExposedIPPrefix"`
	VMDebugMode      bool   `mapstructure:"VMDebugMode"`
	UseSnapshots     bool   `mapstructure:"UseSnapshots"`
	NetworkPoolSize  int    `mapstructure:"NetworkPoolSize"`
}

type DandelionConfig struct {
	DaemonPort int    `mapstructure:"daemonPort"`
	BinaryPath string `mapstructure:"binaryPath"`
	EngineType string `mapstructure:"engineType"`
}

type RedisConf struct {
	Address                  string   `mapstructure:"address"`
	AddressFromDockerNetwork string   `mapstructure:"addressFromDockerNetwork"`
	Password                 string   `mapstructure:"password"`
	Replicas                 []string `mapstructure:"replicas"`
	Db                       int      `mapstructure:"db"`
	FullPersistence          bool     `mapstructure:"fullPersistence"`
}

type AsyncConf struct {
	Enabled           bool `mapstructure:"enabled"`
	PersistRequests   bool `mapstructure:"persistRequests"`
	NumberRetries     int  `mapstructure:"numberRetries"`
	RequestBufferSize int  `mapstructure:"requestBufferSize"`
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

	if workerNodeConfig.CPUConstraints && workerNodeConfig.CRIType != "containerd" {
		logrus.Fatal("Only containerd supports CPU Constraints")
	}

	return workerNodeConfig, nil
}
