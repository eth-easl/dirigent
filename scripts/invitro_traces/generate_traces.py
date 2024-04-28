import os
import json
import shutil

dict_rps_containerd = {
  "Seed": 42,

  "Platform": "Dirigent-RPS",
  "DirigentControlPlaneIP": "10.0.1.253:9092",

  "RpsTarget": 10,
  "RpsColdStartRatioPercentage": 100,
  "RpsCooldownSeconds": 10,
  "RpsImage": "docker.io/cvetkovic/dirigent_empty_function:latest",
  "RpsRuntimeMs": 10,
  "RpsMemoryMB": 2048,
  "RpsIterationMultiplier": 0,

  "TracePath": "data/traces/example",
  "Granularity": "minute",
  "OutputPathPrefix": "data/out/experiment",
  "IATDistribution": "equidistant",
  "CPULimit": "1vCPU",
  "ExperimentDuration": 2,
  "WarmupDuration": 0,

  "IsPartiallyPanic": False,
  "EnableZipkinTracing": False,
  "EnableMetricsScrapping": False,
  "MetricScrapingPeriodSeconds": 15,
  "AutoscalingMetric": "concurrency",

  "GRPCConnectionTimeoutSeconds": 15,
  "GRPCFunctionTimeoutSeconds": 120
}

dict_rps_firecracker = {
  "Seed": 42,

  "Platform": "Dirigent-RPS",

  "DirigentControlPlaneIP": "10.0.1.253:9092",

  "RpsTarget": 10,
  "RpsColdStartRatioPercentage": 100,
  "RpsCooldownSeconds": 10,
  "RpsImage": "empty",
  "RpsRuntimeMs": 10,
  "RpsMemoryMB": 2048,
  "RpsIterationMultiplier": 0,

  "TracePath": "data/traces/example",
  "Granularity": "minute",
  "OutputPathPrefix": "data/out/experiment",
  "IATDistribution": "equidistant",
  "CPULimit": "1vCPU",
  "ExperimentDuration": 2,
  "WarmupDuration": 0,

  "IsPartiallyPanic": False,
  "EnableZipkinTracing": False,
  "EnableMetricsScrapping": False,
  "MetricScrapingPeriodSeconds": 15,
  "AutoscalingMetric": "concurrency",

  "GRPCConnectionTimeoutSeconds": 15,
  "GRPCFunctionTimeoutSeconds": 20
}

script_dir = os.path.dirname(os.path.realpath(__file__))
rps_dir = os.path.join(script_dir, 'rps')

if os.path.isdir(rps_dir):
    shutil.rmtree(rps_dir)

os.mkdir(rps_dir)

for i in range(25,2500, 25):
    obj2 = dict_rps_containerd
    obj2['RpsTarget'] = i

    path = os.path.join(rps_dir, str(i))
    os.mkdir(path)

    with open(os.path.join(path, 'config_containerd.json'), 'w') as fp:
      json.dump(obj2, fp)

    obj3 = dict_rps_firecracker
    obj3['RpsTarget'] = i
    obj3['RpsImage'] = "empty"

    with open(os.path.join(path, 'config_firecracker.json'), 'w') as fp:
      json.dump(obj3, fp)

for i in range(0,11):
    obj2 = dict_rps_containerd
    obj2['RpsTarget'] = i

    path = os.path.join(rps_dir, str(i))
    os.mkdir(path)

    with open(os.path.join(path, 'config_containerd.json'), 'w') as fp:
        json.dump(obj2, fp)

    obj3 = dict_rps_firecracker
    obj3['RpsTarget'] = i
    obj3['RpsImage'] = "empty"

    with open(os.path.join(path, 'config_firecracker.json'), 'w') as fp:
        json.dump(obj3, fp)
