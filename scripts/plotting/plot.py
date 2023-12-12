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
