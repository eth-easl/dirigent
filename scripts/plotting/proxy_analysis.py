import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

def parse_df(path):
    df = pd.read_csv(path)
    df = df[df['cold_start'] == 0]
    df = df.drop(['time', 'service_name', 'container_id'], axis=1)

    df['proxying'] = df['proxying'] - 1460

    return df

async_df = parse_df("proxy_async.csv").mean(axis=0)
sync_df = parse_df("proxy.csv").mean(axis=0)

labels = parse_df("proxy_async.csv").columns.values.tolist()
fig, (ax1,ax2) = plt.subplots(1,2)


ax1.title.set_text("Sync data plane | Total (microseconds) : " + str(int(np.array(sync_df).sum())))
ax1.pie(sync_df, labels=labels,autopct='%1.1f%%')

ax2.title.set_text("Async data plane | Total (microseconds) : " + str(int(np.array(async_df).sum())))
ax2.pie(async_df, labels=labels,autopct='%1.1f%%')


plt.show()