import pandas as pd

from common import *

rootPath = './francois'
load = [100,150,200,250,300,350,400,450]

plotClusteredStackedBarchart(getResult(load, rootPath),
                             title='',
                             clusterLabels=[
                                 '100 colds starts per second',
                                 '150 colds starts per second',
                                 '200 colds starts per second',
                                 '250 colds starts per second',
                                 '300 colds starts per second',
                                 '350 colds starts per second',
                                 '400 colds starts per second',
                                 '450 colds starts per second',
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