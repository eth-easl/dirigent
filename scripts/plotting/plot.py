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

rootPath = '/home/lcvetkovic/Desktop/azure_500/containerd'
load = [500]

# Dirigent
results_to_plot, dirigent_grouped_results = getResult(load, rootPath)

# Knative
knative_results = pd.DataFrame([
    [5],  # [0, 0],
    [1417.12] # [492.69, 383.7]
], columns=['Cluster manager'])  # ['Container creation', 'Readiness probes'])
knative_results.index = ['p50', 'p99']

dirigent_grouped_results.append(knative_results)

plt.figure(figsize=(6, 3))

plotClusteredStackedBarchart(dirigent_grouped_results,
                             title='',
                             clusterLabels=[
                                 'Dirigent',
                                 'Knative-on-K8s',
                             ],
                             clusterLabelPosition=(-1, -0.5), # bars
                             categoryLabelPosition=(0.65, 0.75)) # categories

plt.xlabel('Percentile')
plt.xticks(rotation=0)

plt.ylabel('Function scheduling latency [ms]')
plt.yscale('log')

plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown.png", dpi=160)
plt.savefig(f"{rootPath}/breakdown.pdf", format='pdf', bbox_inches='tight')
