import pandas as pd

cpTracePath = '/home/lcvetkovic/Desktop/cluster_manager/cold_starts/cold_start_trace_1.csv'
proxyTracePath = '/home/lcvetkovic/Desktop/cluster_manager/cold_starts/proxy_trace_1.csv'

cpTrace = pd.read_csv(cpTracePath)
cpTrace['container_id'] = cpTrace['container_id'].astype(str)

proxyTrace = pd.read_csv(proxyTracePath)
proxyTrace['container_id'] = proxyTrace['container_id'].astype(str)

data = pd.merge(proxyTrace, cpTrace, on='container_id', how='inner')

print(data)
