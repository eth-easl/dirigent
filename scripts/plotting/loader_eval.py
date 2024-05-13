import pandas as pd
import sys
import glob
import os
from statsmodels.stats.weightstats import DescrStatsW
from tqdm import tqdm
from scipy import stats
import matplotlib.pyplot as plt
from itertools import cycle

def get_cdf(df, column):
    assert column == 'slowdown' or column == 'waiting_time'
    df = df.drop(labels=df.columns.difference([column]), axis=1)
    # sort by column value
    df.sort_values(by=column, axis='rows', ascending=True)
    # convert to CDF
    if column == 'slowdown' or column == 'waiting_time':
        # we are not using weights for slowdown or waiting time
        df['density'] = 1 / df.shape[0]
    df = df.groupby(column).sum()
    res = df['density'].cumsum()
    # make sure CDF starts from zero
    index = res.index[0]
    if index > -1:
        res[index] = 0.0
    return res


def process_latency(path):
    df = pd.read_csv(path)
    print(f"{len(df)} rows before dropping warmup")
    df['time'] = df['startTime'] - df['startTime'][0]
    warmupDurationMicrosecondss = warmupDurationMs * 1e3
    df = df.loc[df['time'] > warmupDurationMicrosecondss]
    print(f"{len(df)} rows after dropping warmup")
    df = df[df['functionTimeout'] == False]
    df = df[df['connectionTimeout'] == False]
    df = df[df['memoryAllocationTimeout'] == False]
    print(f"{len(df)} rows after dropping timeouts")
    idx = 2
    if 'oracle' or 'predictor'  in path:
        idx = 1
    print("check what instance name is to change idx on line 44")
    idx = 1
    df['function_hash'] = df['instance']
    print(f"Number of functios {len(df['function_hash'].unique())}")
    df = df.reset_index(drop=True)
    df['slowdown'] = df['responseTime'] / df['actualDuration']
    df = df.groupby(df.function_hash).slowdown.apply(stats.gmean)
    df = df.to_frame()
    cdf = get_cdf(df, 'slowdown')
    return df['slowdown'].describe(percentiles=[.50, .95, .99]), cdf, stats.gmean(df['slowdown'])


def slowdown_over_time(path):
    df = pd.read_csv(path)
    df['time'] = df['startTime'] - df['startTime'][0]
    # warmupDurationMicrosecondss = warmupDurationMs * 1e3
    # df = df.loc[df['time'] > warmupDurationMicrosecondss]
    df = df[df['functionTimeout'] == False]
    df = df[df['connectionTimeout'] == False]
    df = df[df['memoryAllocationTimeout'] == False]
    idx = 2
    if 'oracle' or 'predictor' in path:
        idx = 1
    print("check what instance name is to change idx on line 63")
    idx = 1
    df['function_hash'] = df['instance']
    print(f"Number of functios {len(df['function_hash'].unique())}")
    df = df.reset_index(drop=True)
    df['slowdown'] = df['responseTime'] / df['actualDuration']

    # for each minute:
    # group by function applying geo mean
    # then apply geo mean on all of those
    # get 1 point per minute
    df['minute'] = (df['time']/(60*1000*1000)).astype(int)
    df = df[['minute', 'slowdown', 'function_hash']]
    df = df.reset_index(drop=True)
    df = pd.pivot_table(df, index=['minute', 'function_hash'], aggfunc=stats.gmean).reset_index()
    df = df.groupby(df.minute, as_index=False).slowdown.apply(stats.gmean)
    return df


def plot_cdfs(metric):
    plt.figure()
    d = {}
    if metric == 'Slowdown':
        d = slowdown_cdfs
    for experiment, data in d.items():
        df = data
        df = df.reset_index()
        x = df.iloc[:, 0]
        y = df.iloc[:, 1]
        plt.step(x, y, label=experiment, where="post")
    plt.grid()
    plt.legend()
    x_axis_label = metric
    plt.xlabel(x_axis_label)
    plt.ylabel("CDF")
    plt.ylim([0, 1])
    plt.title(metric + " cdf")
    plt.xscale("log")
    plt.savefig(outputFolder + "/" + metric + "_cdf.png")

def plot_over_time(metric):
    plt.figure()
    d = {}
    if metric == 'Slowdown':
        d = slowdownOverTimeDfs
    for experiment, data in d.items():
        plt.plot(data['minute'], data['slowdown'], label=experiment)
    plt.grid()
    plt.legend()
    plt.xlabel("time in minutes")
    plt.ylabel("Slowdown")
    plt.ylim(1,200)
    plt.title("Slowdown over Time")
    plt.savefig(outputFolder + "/slowdown_over_time.png")


if len(sys.argv) < 4:
    print("Invalid arguments provided to statistics script.")
    exit(1)
outputFolder = sys.argv[1]
print(f"output folder is set to {outputFolder}")
if not os.path.exists(outputFolder):
    os.makedirs(outputFolder)
warmupDuration = sys.argv[2]
print(f"warmup duration is set to {warmupDuration} minutes")
warmupDurationMs = int(warmupDuration) * 60 * 1e3
slowdownDf = pd.DataFrame(columns=['Experiment', 'mean', 'p50', 'p95', 'p99'])
slowdown_cdfs = {}
slowdownOverTimeDfs = {}

for i in sys.argv[3:]:  # first three values in sys.argv are not input data
    experimentName = i.split("/")
    expName = experimentName[0]
    for c in experimentName[1:]:
        if len(c) != 0:
            expName = expName + "_" + c
    latencyLog = glob.glob(i + "*.csv")
    name = latencyLog[0].removesuffix(".csv")
    assert len(latencyLog) == 1, f"Expected 1 latency log, got {len(latencyLog)} logs"
    latencyLog = latencyLog[0]
    slowdown_over_time_df = slowdown_over_time(latencyLog)
    slowdownStats, latency_cdf, geo_mean = process_latency(latencyLog)
    slowdownStats['Experiment'] = expName
    slowdownStats['mean'] = geo_mean
    slowdownStats = slowdownStats.drop(['count', 'std', 'min', 'max'])
    slowdownStats = slowdownStats.tolist()
    slowdownStats = [slowdownStats[-1]] + slowdownStats[:-1]
    slowdownDf.loc[len(slowdownDf)] = slowdownStats
    slowdown_cdfs[expName] = latency_cdf
    slowdownOverTimeDfs[expName] = slowdown_over_time_df

slowdownDf = slowdownDf.round(decimals=2)


slowdownDf.to_html(outputFolder + "/slowdown_df.html", index=False)
# colors for bar plots as well as the legend:
newColors = []
cycol = cycle('bgrcmk')
for i in range(len(slowdownDf)):
    color = next(cycol)
    newColors.append(color)
colors = {}
for idx, k in enumerate(slowdown_cdfs.keys()):
    colors[k] = newColors[idx]
labels = list(colors.keys())
handles = [plt.Rectangle((0, 0), 1, 1, color=colors[label]) for label in labels]
plt.legend(handles, labels, loc='center left', bbox_to_anchor=(1, 0.5))

cdfs_to_plot = ['Slowdown']
for metric in cdfs_to_plot:
    plot_cdfs(metric)
    plot_over_time(metric)
