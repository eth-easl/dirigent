import pandas as pd

from common import *

rootPath = './francois'
load = [800]

plotClusteredStackedBarchart(getResult(load, rootPath),
                             title='Test plot',
                             clusterLabels=[
                                 'x cold start',
                                 'x cold start',
                                 'x cold start',
                                 'x cold start',
                                 'x cold start',
                                 'x cold start',
                                 'x cold start',
                                 'x cold start',
                             ],
                             clusterLabelPosition=(-0.335, 1.1),
                             categoryLabelPosition=(-0.25, 0.65))

plt.title(f'Cold start latency breakdown burst')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown.png",dpi=160)