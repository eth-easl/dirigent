import glob

import matplotlib.pyplot as plt
import numpy
import pandas as pd


def index_result_files(path):
    files = []
    for file in glob.glob(f"{path}/*.csv"):
        files.append(file)

    return files


def get_processing_list(path):
    allFiles = index_result_files(path)

    xPoints = []
    toProcess = []

    for f in allFiles:
        rpsStart = f.rfind("rps_")
        rpsEnd = f.rfind(".")

        if rpsStart != -1:
            rps = f[rpsStart + 4: rpsEnd]

            xPoints.append(int(rps))

    xPoints.sort()

    for f in xPoints:
        toProcess.append(f"{path}/rps_{f}.csv")

    return xPoints, toProcess


def get_knative_data():
    xPoints, toProcess = get_processing_list("cold_start_sweep/knative/results/")

    if len(xPoints) == 0:
        return [], [], []

    p50_measurements, p50_std = [], []
    p99_measurements, p99_std = [], []
    warm_start_ratio = []

    for f in toProcess:
        df = pd.read_csv(f)

        warmStartRatio = len(df[df.responseTime < 300_000].index) / len(df.index)
        print(f"Ratio of warm starts before filtering: {warmStartRatio}")

        # Filter out warmup phase and warm starts
        df = df[(df.phase == 2) & (df.responseTime > 300_000)]
        print("Sample size after filtering: ", len(df.index))

        p50 = df['responseTime'].quantile(0.5) / 1000  # ms
        p99 = df['responseTime'].quantile(0.99) / 1000  # ms
        p50_measurements.append(p50)
        p99_measurements.append(p99)
        warm_start_ratio.append(warmStartRatio)

        print("p50: ", p50, "ms")
        print("p99: ", p99, "ms")

        print()

    return xPoints, p50_measurements, p99_measurements


def get_dirigent_data(technology):
    x_values, files = get_processing_list(f"cold_start_sweep/dirigent/results_{technology}/")

    dirigent_y_p50 = []
    dirigent_y_p99 = []

    for rps in files:
        df = pd.read_csv(rps)

        p50 = df['responseTime'].quantile(0.5) / 1000  # ms
        p99 = df['responseTime'].quantile(0.99) / 1000  # ms

        dirigent_y_p50.append(p50)
        dirigent_y_p99.append(p99)

    return x_values, dirigent_y_p50, dirigent_y_p99


def cold_start_sweep():
    fig, ax1 = plt.subplots()

    knative_xPoint, knative_p50_measurements, knative_p99_measurements = get_knative_data()
    if len(knative_xPoint) != 0:
        ax1.plot(knative_xPoint, knative_p50_measurements, color='tab:blue', marker='x', label='Knative - p50')
        ax1.plot(knative_xPoint, knative_p99_measurements, color='tab:blue', marker='x', linestyle='dashed',
                 label='Knative - p99')

    dirigent_containerd_xPoint, dirigent_containerd_p50_measurements, dirigent_containerd_p99_measurements = get_dirigent_data(
        'containerd')
    if len(dirigent_containerd_xPoint) != 0:
        ax1.plot(dirigent_containerd_xPoint, dirigent_containerd_p50_measurements, color='tab:orange', marker='o',
                 label=f'Dirigent - containerd - p50')
        ax1.plot(dirigent_containerd_xPoint, dirigent_containerd_p99_measurements, color='tab:orange', marker='o',
                 linestyle='dashed',
                 label=f'Dirigent - containerd - p99')

    dirigent_firecracker_xPoint, dirigent_firecracker_p50_measurements, dirigent_firecracker_p99_measurements = get_dirigent_data(
        'firecracker')
    if len(dirigent_firecracker_xPoint) != 0:
        ax1.plot(dirigent_firecracker_xPoint, dirigent_firecracker_p50_measurements, color='tab:green', marker='s',
                 label=f'Dirigent - Firecracker - p50')
        ax1.plot(dirigent_firecracker_xPoint, dirigent_firecracker_p99_measurements, color='tab:green', marker='s',
                 linestyle='dashed',
                 label=f'Dirigent - Firecracker - p99')

    ax1.set_xlabel('Cold starts per second')
    ax1.set_ylabel('End-to-end latency [ms]')

    ax1.set_ylim([0, 5_000])
    ax1.legend()

    plt.xscale('log')
    # plt.yscale('log')
    plt.ylim([10, 5_000])

    plt.legend(loc='upper center')

    plt.grid()
    plt.tight_layout()
    plt.savefig(f"cold_start_sweep.png")

cold_start_sweep()
