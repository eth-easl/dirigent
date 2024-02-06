import pandas as pd
import matplotlib.pyplot as plt
import numpy as np

df = pd.read_csv("proxy.csv")

df = df[df['cold_start'] == 0]

df = df.drop(['time', 'service_name', 'container_id', 'proxying'], axis=1)

print(df.head(20))

labels = df.columns.values.tolist()
sizes = df.mean(axis=0)

fig, ax = plt.subplots()
ax.pie(sizes, labels=labels,autopct='%1.1f%%')

print(sizes)


plt.title("Percentage spent in each part of the proxy")
plt.show()