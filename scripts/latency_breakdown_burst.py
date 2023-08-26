import pandas as pd

from common import *

rootPath = './francois'
load = [50,100,200,400,800, 900, 1000,1600]

plotClusteredStackedBarchart(getResult(load, rootPath),
                             title='',
                             clusterLabels=[
                                 '50 colds starts',
                                 '100 colds starts',
                                 '200 colds starts',
                                 '400 colds starts',
                                 '800 colds starts',
                                 '900 colds starts',
                                 '1000 colds starts',
                                 '1600 colds starts',
                             ],
                             clusterLabelPosition=(-0.335, 1.1),
                             categoryLabelPosition=(-0.25, 0.65))

plt.title(f'Cold start latency breakdown burst')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_burst.png",dpi=160)