from common import *

rootPath = '/home/lcvetkovic/Desktop/data_analysis'
load = [500]

plotClusteredStackedBarchart(getResult(load, rootPath),
                             title='',
                             clusterLabels=[
                                 'Azure 500 functions',
                             ],
                             clusterLabelPosition=(-0.20, 1.1),
                             categoryLabelPosition=(-0.25, 0.65))

plt.title(f'Cold start latency breakdown burst')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_burst.png", dpi=160)
plt.savefig(f"{rootPath}/breakdown_burst.pdf", dpi=160)
