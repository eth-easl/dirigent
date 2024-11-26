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

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

load = ['./output/osdi', './output/main']


percent = np.arange(1, 101, 1)

result = []
for rootPath in load:
    cpTrace = pd.read_csv(f'{rootPath}/cold_start_trace_500.csv')
    proxyTrace = pd.read_csv(f'{rootPath}/proxy_trace_500.csv')

    data = pd.merge(proxyTrace, cpTrace, on=['container_id', 'service_name'], how='inner')
    data = data[data['success'] == True]  # keep only successful invocations
    data = data[data['cold_start'] > 0]  # keep only cold starts

    data = data.drop(columns=['time_x', 'time_y', 'success', 'service_name', 'container_id'])

    data['control_plane'] = data['cold_start'] - \
                            (data['image_fetch'] + data['sandbox_create'] + data['sandbox_start'] +
                             data['network_setup'] + data['iptables'] + data['other_worker_node'])
    data = data.drop(columns=['cold_start'])

    hist = []
    for per in percent:
        hist.append(np.sum(data.quantile(per / 100)))
    plt.plot(hist, percent,  label="{} colds starts per second".format(l))




plt.title(f'CDF - Sweep test')
plt.ylabel('Percentile')
plt.xlabel('Latency [ms]')
plt.legend()
plt.savefig(f"{rootPath}/cdf_sweep.png",dpi=160)