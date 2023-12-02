import matplotlib.pyplot as plt
import numpy as np
import pandas as pd


def processQuantile(d, percentile):
    p = d.reset_index()
    p = p.T
    p.columns = p.iloc[0]
    p = p.iloc[1:, :]
    p = p.rename(index={percentile: f"p{int(percentile * 100)}"})

    return p


def getResult(load, rootPath):
    result = []
    for l in load:
        cpTrace = pd.read_csv(f'{rootPath}/cold_start_trace_{l}.csv')
        proxyTrace = pd.read_csv(f'{rootPath}/proxy_trace_{l}.csv')

        # filter first 10 minutes
        minimalTimestamp = proxyTrace['time'].min() # nanoseconds
        minimalTimestamp += 60 * 10e9 # 10 minutes

        proxyTrace = proxyTrace[proxyTrace['time'] >= minimalTimestamp]

        data = pd.merge(proxyTrace, cpTrace, on=['container_id', 'service_name'], how='inner')
        data = data[data['success'] == True]  # keep only successful invocations

        data = data.drop(columns=['time_x', 'time_y', 'success', 'service_name', 'container_id'])

        data['control_plane_other'] = data['cold_start'] - \
                                (data['image_fetch'] + data['sandbox_create'] + data['sandbox_start'] +
                                 data['network_setup'] + data['iptables'] + data['readiness_probe'] +
                                 data['snapshot_creation'] + data['configure_monitoring'] +
                                 data['find_snapshot'] + data['other_worker_node'])

        # make control plane overhead for non-cold starts be zero
        data.loc[data['cold_start'] == 0, "control_plane_other"] = 0
        data = data[data['control_plane_other'] >= 0]

        # drop column that was broken down
        data = data.drop(columns=['cold_start'])

        p50 = data.quantile(0.5)
        p99 = data.quantile(0.99)

        dataToPlot = processQuantile(p50, 0.5)
        dataToPlot = pd.concat([dataToPlot, processQuantile(p99, 0.99)])
        dataToPlot = dataToPlot / 1000  # μs -> ms

        # drop queueing and user code execution
        dataToPlot = dataToPlot.drop(columns=['cc_throttling', 'proxying'])

        dataToPlot = dataToPlot.rename(columns={
            "get_metadata": "DP - find function",
            "add_deployment": "DP - update endpoints",
            "load_balancing": "DP - load balancing",
            "cc_throttling": "DP - cc. throttling",
            "proxying": "DP - proxying",
            "other": "DP - other",
            "image_fetch": "WN - image fetch",
            "sandbox_create": "WN - sandbox create",
            "sandbox_start": "WN - sandbox start",
            "network_setup": "WN - get network",
            "iptables": "WN - iptables",
            "readiness_probe": "WN - readiness",
            "snapshot_creation": "WN - create snapshot",
            "configure_monitoring": "WN - configure monitors",
            "find_snapshot": "WN - get snapshot",
            "other_worker_node": "WN - other",
            "data_plane_propagation": "CP - propagate endpoints",
            "control_plane_other": "CP - rest",
        })

        result.append(dataToPlot)

    return result


# Taken from https://stackoverflow.com/questions/22787209/how-to-have-clusters-of-stacked-bars
def plotClusteredStackedBarchart(dataToPlot,
                                 clusterLabels=None,
                                 clusterLabelPosition=(0.1, -0.05),
                                 categoryLabelPosition=(1.0, -0.05),
                                 title="multiple stacked bar plot", H="/",
                                 **kwargs):
    n_df = len(dataToPlot)
    n_col = len(dataToPlot[0].columns)
    n_ind = len(dataToPlot[0].index)
    axe = plt.subplot(111)

    for df in dataToPlot:  # for each data frame
        axe = df.plot(kind="bar",
                      linewidth=0,
                      stacked=True,
                      ax=axe,
                      legend=False,
                      grid=False,
                      **kwargs)  # make bar plots

    subtractFromXOffset = 0
    if n_df <= 2:
        subtractFromXOffset = 0.10
    elif n_df <= 4:
        subtractFromXOffset = 0.15
    elif n_df <= 8:
        subtractFromXOffset = 0.2
    elif n_df <= 16:
        subtractFromXOffset = 0.22
        H = ''
    else:
        subtractFromXOffset = 0.23
        H = ''

    h, l = axe.get_legend_handles_labels()  # get the handles we want to modify
    for i in range(0, n_df * n_col, n_col):  # len(h) = n_col * n_df
        for j, pa in enumerate(h[i:i + n_col]):
            for rect in pa.patches:  # for each index
                rect.set_x(rect.get_x() + 1 / float(n_df + 1) * i / float(
                    n_col) - subtractFromXOffset)  # for 8 clusters subtract 0.15
                rect.set_hatch(H * int(i / n_col))  # edited part
                rect.set_width(1 / float(n_df + 1))

    axe.set_xticks((np.arange(0, 2 * n_ind, 2) + 1 / float(n_df + 1)) / 2.)
    axe.set_xticklabels(df.index, rotation=0)
    axe.set_title(title)

    # Add invisible data to add another legend
    n = []
    for i in range(n_df):
        n.append(axe.bar(0, 0, color="gray", hatch=H * i))

    if categoryLabelPosition is not None:
        l1 = axe.legend(h[:n_col], l[:n_col], ncol=1, bbox_to_anchor=categoryLabelPosition)
        axe.add_artist(l1)

    if clusterLabels is not None and n_df <= 8 and clusterLabelPosition is not None:
        plt.legend(n, clusterLabels, bbox_to_anchor=clusterLabelPosition)

    return axe
