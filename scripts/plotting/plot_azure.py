import matplotlib.pyplot as plt
import pandas as pd
from scipy import stats


path = "/home/francois/Documents/cluster_manager/scripts/plotting"

def cdf(df, column):
    df['density'] = 1 / df.shape[0]
    df = df.groupby(column).sum(numeric_only=True)

    res = df['density'].cumsum()

    # make sure CDF starts from zero
    index = res.index[0]
    if index > 0:
        res[index] = 0.0

    return res


def getCurve(path):
    df = pd.read_csv(path)

    # discard first 10 minutes
    df = df[~df.invocationID.str.contains("^min[0-1]\.", regex=True)]

    print(f"Failure percentage: {df[df['connectionTimeout'] | df['functionTimeout']].shape[0] / df.shape[0]}")

    df['function_hash'] = df['instance'].str.split("-").str[2]
    df = df.reset_index(drop=True)
    df['slowdown'] = df['responseTime'] / df['requestedDuration']
    print(f"Slowdown < 1: {df[df['slowdown'] < 1].shape[0] / df.shape[0]}")

    # number of instances created
    print(f"Number of instances created: {df['instance'].nunique()}")

    # filter functions with invalid slowdown
    df = df[df['slowdown'] >= 1]

    df = df.groupby(df.function_hash).slowdown.apply(stats.gmean)
    df = df.to_frame()

    # filter warm starts
    # beforeWarmFilter = df.shape[0]
    # df = df[df['actualDuration'] / df['responseTime'] <= 0.8]
    # afterWarmFilter = df.shape[0]
    # print(f"Warm start ratio: {(beforeWarmFilter - afterWarmFilter) / beforeWarmFilter}")

    print()

    return cdf(df, 'slowdown')


off = getCurve('azure_100.csv')
#no_endpoint = getCurve('test_persistence/azure_501.csv')
#endpoint = getCurve('test_persistence/azure_502.csv')


plt.figure(figsize=(6, 3))

plt.plot(off, label='Dirigent vanilla - docker-compose DOWN')


plt.xlabel("Per-function slowdown")
plt.ylabel("CDF")

# plt.xlim([0, 100])
# plt.ylim([0.8, 1])

plt.xscale('log')

plt.legend()
plt.grid()

plt.tight_layout()

plt.savefig(f"{path}/combined/azure_500.png")
plt.savefig(f"{path}/combined/azure_500.pdf", format='pdf', bbox_inches='tight')