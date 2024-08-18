import os.path

import matplotlib.pyplot as plt
import pandas as pd
from scipy import stats

real_path = os.path.dirname(os.path.realpath(__file__))

knative_path = f'{real_path}/azure_500/knative/results_azure_500/experiment_duration_30.csv'
dirigent_containerd = f'{real_path}/azure_500/dirigent/results_azure_500/experiment_duration_30.csv'
dirigent_firecracker = f'{real_path}/azure_500/dirigent/results_azure_500_firecracker/experiment_duration_30.csv'


def cdf(df, column):
    df['density'] = 1 / df.shape[0]
    df = df.groupby(column).sum(numeric_only=True)

    res = df['density'].cumsum()

    # make sure CDF starts from zero
    index = res.index[0]
    if index > 0:
        res[index] = 0.0

    return res


def getCurve(path, idx=0):
    df = pd.read_csv(path)

    # discard first 10 minutes
    df = df[~df.invocationID.str.contains("^min[0-9]\.", regex=True)]

    print(path)
    print(f"Failure percentage: {df[df['connectionTimeout'] | df['functionTimeout']].shape[0] / df.shape[0]}")

    df['function_hash'] = df['instance'].str.split("-").str[idx]
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

    print(path)
    print(df.sort_values(by='slowdown', ascending=False).head(10))

    print()

    return cdf(df, 'slowdown')


def plot_per_function_slowdown():
    plt.figure(figsize=(8, 4))

    if os.path.exists(knative_path):
        plt.plot(getCurve(knative_path), label='Knative', color='tab:blue', linestyle='dotted')
    if os.path.exists(dirigent_containerd):
        plt.plot(getCurve(dirigent_containerd), label='Dirigent - containerd', color='tab:orange')
    if os.path.exists(dirigent_firecracker):
        plt.plot(getCurve(dirigent_firecracker), label='Dirigent - Firecracker', color='tab:green')

    #########################
    #########################
    #########################

    plt.xlabel("Per-function slowdown")
    plt.ylabel("CDF")

    plt.xscale('log')

    plt.xlim([1, 10e5])

    plt.legend()
    plt.grid()

    plt.tight_layout()

    plt.savefig("azure_500_per_function_slowdown.png")


def plot_function_scheduling_latency():
    def get_per_invocation(path):
        df = pd.read_csv(path)

        df['responseTime'] -= df['actualDuration']
        df['responseTime'] /= 1000  # to ms

        return cdf(df, 'responseTime')

    def get_per_function(path):
        df = pd.read_csv(path)

        df['responseTime'] -= df['actualDuration']
        df['responseTime'] /= 1000  # to ms

        df = df.apply(lambda x: x.replace({'(\-[0-9]+(\-[0-9]+\-deployment\-[0-9A-Za-z\-]+){0,1})$': ""}, regex=True))

        test = df.groupby('instance')['responseTime'].mean().to_frame()

        return cdf(test, 'responseTime')

    fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(8, 4))

    if os.path.exists(knative_path):
        ax1.plot(get_per_invocation(knative_path), label='Knative', color='tab:blue', linestyle='dotted')
        ax2.plot(get_per_function(knative_path), label='Knative', color='tab:blue', linestyle='dotted')

    if os.path.exists(dirigent_containerd):
        ax1.plot(get_per_invocation(dirigent_containerd), label='Dirigent', color='tab:green')
        ax2.plot(get_per_function(dirigent_containerd), label='Dirigent', color='tab:green')

    ax1.set_xlabel('Per-invocation scheduling\nlatency [ms]')
    ax1.set_ylabel('CDF')
    ax1.set_xscale('log')
    ax1.set_xticks([1, 10, 100, 1_000, 10_000, 100_000, 1_000_000])
    ax1.set_xlim([1, 10 ** 6])
    ax1.grid()
    ax1.legend()

    ax2.set_xlabel('Average per-function\nscheduling latency [ms]')
    ax2.set_ylabel('CDF')
    ax2.set_xscale('log')
    ax2.set_xticks([1, 10, 100, 1_000, 10_000, 100_000, 1_000_000])
    ax2.set_xlim([1, 10 ** 6])
    ax2.grid()
    ax2.legend()

    fig.set_figheight(4)
    fig.set_figwidth(8)

    plt.tight_layout()

    plt.savefig("azure_500_scheduling_latency.png")


plot_per_function_slowdown()
plot_function_scheduling_latency()
