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

from common import *

rootPath = '/home/lcvetkovic/Desktop/replay/sosp/dirigent_1cp_1dp/rps_sweep_containerd/containerd'
load = [250, 1000, 1250]

#load = [500, 2000, 2500]

labels = []
[labels.append(f'{x} CS/s') for x in load]

rawData = getResult(load, rootPath)[0]
for idx in range(0, len(labels)):
    rawData[idx].to_csv(f'{rootPath}/statistics_load_{load[idx]}.csv')

plt.figure(figsize=(3, 6))
plotClusteredStackedBarchart(rawData,
                             title='',
                             clusterLabels=labels,
                             clusterLabelPosition=(0.15, 0.1),
                             categoryLabelPosition=(0.35, 0.65))

plt.title(f'Cold start latency\nbreakdown sweep')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_sweep.png", dpi=160)
