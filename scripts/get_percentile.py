import pandas as pd

data = pd.read_csv("/home/lcvetkovic/Desktop/cluster_manager/cold_start_steady_load/experiment_50rps.csv")

data = data[data['responseTime'] > 50_000]
p50 = data[['responseTime']].quantile(0.5)
p95 = data[['responseTime']].quantile(0.95)

print(p50)
print(p95)