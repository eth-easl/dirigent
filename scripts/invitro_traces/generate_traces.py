#  MIT License
#
#  Copyright (c) 2024 EASL
#
#  Permission is hereby granted, free of charge, to any person obtaining a copy
#  of this software and associated documentation files (the "Software"), to deal
#  in the Software without restriction, including without limitation the rights
#  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
#  copies of the Software, and to permit persons to whom the Software is
#  furnished to do so, subject to the following conditions:
#
#  The above copyright notice and this permission notice shall be included in all
#  copies or substantial portions of the Software.
#
#  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
#  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
#  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
#  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
#  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
#  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
#  SOFTWARE.

import numpy as np
import pandas as pd
import glob
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

if os.path.isdir("rps"):
    shutil.rmtree('rps')

os.mkdir("rps")

for i in range(25,2500, 25):
    obj2 = dict_rps_containerd
    obj2['RpsTarget'] = i

    path = "rps/" + str(i)
    os.mkdir(path)

    with open(path + '/config_containerd.json', 'w') as fp:
      json.dump(obj2, fp)

    obj3 = dict_rps_firecracker
    obj3['RpsTarget'] = i
    obj3['RpsImage'] = "empty"

    with open(path + '/config_firecracker.json', 'w') as fp:
      json.dump(obj3, fp)

for i in range(0,11):
    obj2 = dict_rps_containerd
    obj2['RpsTarget'] = i

    path = "rps/" + str(i)
    os.mkdir(path)

    with open(path + '/config_containerd.json', 'w') as fp:
        json.dump(obj2, fp)

    obj3 = dict_rps_firecracker
    obj3['RpsTarget'] = i
    obj3['RpsImage'] = "empty"

    with open(path + '/config_firecracker.json', 'w') as fp:
        json.dump(obj3, fp)