import glob
import sys

import matplotlib.pyplot as plt
import pandas as pd

input_folder_knative = sys.argv[1]
input_folder_dirigent = sys.argv[2]
output_folder = sys.argv[3]

fig, (ax1, ax2) = plt.subplots(2, 1, sharex=True, figsize=(8, 6))


def plot_experiment(experiment_name, input_folder, column):
    nodes = glob.glob(input_folder + "/cpu_mem_usage/*.csv")
    for n in nodes:
        if experiment_name == "Knative-on-K8s" and "hp058" in n:
            master_node = n
            nodes.remove(n)
        if experiment_name == "Knative-on-K8s" and "hp075" in n:
            nodes.remove(n)
        if experiment_name == "Dirigent" and "hp075" in n:
            master_node = n
            nodes.remove(n)
        if experiment_name == "Dirigent" and "hp058" in n:
            nodes.remove(n)
        if experiment_name == "Dirigent" and "hp156" in n:
            nodes.remove(n)
    experiment_df = pd.read_csv(input_folder + "/experiment_duration_30.csv")
    master_node_df = pd.read_csv(master_node)
    start = experiment_df['startTime'][0] / 1e6
    end = experiment_df['startTime'].iloc[-1] / 1e6

    master_node_df = master_node_df[master_node_df['Timestamp'] > start]
    master_node_df = master_node_df[master_node_df['Timestamp'] < end]
    master_node_df = master_node_df.reset_index(drop=True)

    master_node_df['time'] = master_node_df['Timestamp'] - master_node_df['Timestamp'][0]
    master_node_df['minute'] = master_node_df['time'] / 60
    master_node_df = master_node_df[master_node_df['minute'] >= 10]

    master_node_df['minute'] = (master_node_df['minute']).round(0).astype('int')
    master_node_df = master_node_df.groupby(master_node_df.minute, as_index=False).mean()

    # need to use space before column to access it...
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


column = ' CPUUtilization'
# column=' memoryUtilization '

plot_experiment("Knative-on-K8s", input_folder_knative, column=column)
plot_experiment("Dirigent", input_folder_dirigent, column=column)

ax1.set_title("Master Node")
ax1.set_ylabel("CPU Utilization [%]")
ax1.set_ylim(0, 100)
#ax1.set_xlabel("Time [min]")
ax1.grid()
ax1.legend()

ax2.set_title("Worker Nodes")
ax2.set_ylabel("CPU Utilization [%]")
ax2.set_ylim(0, 100)
ax2.set_xlabel("Time [min]")
ax2.grid()
ax2.legend()

plt.savefig(f"{output_folder}/azure_500_CPU_utilization.png")
plt.savefig(f"{output_folder}/azure_500_CPU_utilization.pdf", format='pdf', bbox_inches='tight')
