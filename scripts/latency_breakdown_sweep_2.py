import pandas as pd

from common import *

rootPath = './francois'
load = [500,550,600,650,700,750,800,850]

plotClusteredStackedBarchart(getResult(load, rootPath),
                             title='',
                             clusterLabels=[
                                 '500 colds starts per second',
                                 '550 colds starts per second',
                                 '600 colds starts per second',
                                 '650 colds starts per second',
                                 '700 colds starts per second',
                                 '750 colds starts per second',
                                 '800 colds starts per second',
                                 '850 colds starts per second',
                             ],
                             clusterLabelPosition=(-0.15, 1.1),
                             categoryLabelPosition=(-0.35, 0.65))

plt.title(f'Cold start latency breakdown burst')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_sweep.png",dpi=160)