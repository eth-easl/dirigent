import glob

import matplotlib.pyplot as plt
import pandas as pd

input_folder_knative = '/home/lcvetkovic/Desktop/replay/sosp/knative_3cp_3dp/azure_500'
input_folder_dirigent = '/home/lcvetkovic/Desktop/replay/sosp/dirigent_3cp_3dp/azure_500'

output_folder = '/home/lcvetkovic/Desktop/replay/sosp/dirigent_3cp_3dp/azure_500'


def plot_experiment(experiment_name, input_folder, column):
    nodes = glob.glob(input_folder + "/cpu_mem_usage/*.csv")
    master_nodes = []

    for n in nodes:
        # Master node
        if experiment_name == "Knative-on-K8s" and ("hp156" in n or "hp091" in n or "hp155" in n): #023
            master_nodes.append(n)
            nodes.remove(n)

        # Loader node
        if experiment_name == "Knative-on-K8s" and "hp004" in n: #075
            nodes.remove(n)

        # Master node(s)
        if experiment_name == "Dirigent" and ("hp023" in n):# or "hp091" in n or "hp081" in n):
            master_nodes.append(n)
            nodes.remove(n)

        # Data plane(s)
        if experiment_name == "Dirigent" and ("hp091" in n): # or "hp077" in n or "hp134" in n):
            nodes.remove(n)

        # Loader node
        if experiment_name == "Dirigent" and ("hp080" in n):
            nodes.remove(n)

    experiment_df = pd.read_csv(input_folder + "/experiment_duration_30.csv")
    start = experiment_df['startTime'][0] / 1e6
    end = experiment_df['startTime'].iloc[-1] / 1e6

    id = 0
    master_node_df = pd.DataFrame()
    for n in master_nodes:
        df = pd.read_csv(n)
        df = df[df['Timestamp'] > start]
        df = df[df['Timestamp'] < end]
        df = df.reset_index(drop=True)

        df['time'] = df['Timestamp'] - df['Timestamp'][0]
        df['minute'] = df['time'] / 60
        df = df[df['minute'] >= 10]

        df['minute'] = (df['minute']).round(0).astype('int')
        df = df.groupby(df.minute, as_index=False).mean()

        df['id'] = id
        if id == 0:
            master_node_df = df
        id += 1
        master_node_df = pd.concat([master_node_df, df], ignore_index=True)

    # need to use space before column to access it...
    master_node_df = master_node_df.groupby(master_node_df.minute, as_index=False).mean()
    ax1.step(master_node_df['minute'], master_node_df[column], label=experiment_name, where='post')

    id = 0
    worker_df = pd.DataFrame()
    for n in nodes:
        df = pd.read_csv(n)
        df = df[df['Timestamp'] > start]
        df = df[df['Timestamp'] < end]
        df = df.reset_index(drop=True)

        df['time'] = df['Timestamp'] - df['Timestamp'][0]
        df['minute'] = df['time'] / 60
        df = df[df['minute'] >= 10]

        df['minute'] = (df['minute']).round(0).astype('int')
        df = df.groupby(df.minute, as_index=False).mean()
        df['id'] = id
        if id == 0:
            worker_df = df
        id += 1
        worker_df = pd.concat([worker_df, df], ignore_index=True)

    worker_df = worker_df.groupby(worker_df.minute, as_index=False).mean()
    ax2.step(worker_df['minute'], worker_df[column], label=experiment_name, where='post')


for column in [' CPUUtilization', ' memoryUtilization ']:
    fig, (ax1, ax2) = plt.subplots(2, 1, sharex=True, figsize=(8, 6))

    plot_experiment("Knative", input_folder_knative, column=column)
    plot_experiment("Dirigent", input_folder_dirigent, column=column)

    ax1.set_title("Master Nodes")
    if column == ' CPUUtilization':
        ax1.set_ylabel("CPU Utilization [%]")
    else:
        ax1.set_ylabel("Memory Utilization [%]")
    ax1.set_ylim(0, 100)
    # ax1.set_xlabel("Time [min]")
    ax1.grid()
    ax1.legend()

    ax2.set_title("Worker Nodes")

    if column == ' CPUUtilization':
        ax2.set_ylabel("CPU Utilization [%]")
    else:
        ax2.set_ylabel("Memory Utilization [%]")
    ax2.set_ylim(0, 100)
    ax2.set_xlabel("Time [min]")
    ax2.grid()
    ax2.legend()

    plt.savefig(f"{output_folder}/{column.strip()}.png")
    plt.savefig(f"{output_folder}/{column.strip()}.pdf", format='pdf', bbox_inches='tight')
