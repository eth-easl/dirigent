# Taken from

import matplotlib.pyplot as plt
import numpy as np

sizes = ("1", "10", "100", "1000")
data = {
    'No persistence': (1,10,84,808),
    'Redis - default mode': (3,13,131,1286),
    'Redis - full persistence': (3,18,174,1730),
}

x = np.arange(len(sizes))  # the label locations
width = 0.25  # the width of the bars
multiplier = 0

fig, ax = plt.subplots(layout='constrained')

for attribute, measurement in data.items():
    offset = width * multiplier
    rects = ax.bar(x + offset, measurement, width, label=attribute)
    ax.bar_label(rects, padding=3)
    multiplier += 1

# Add some text for labels, title and custom x-axis tick labels, etc.
ax.set_ylabel('Latency in milliseconds', fontsize=20)
ax.set_xticks(x + width, sizes)
ax.legend(loc='upper left',fontsize=15)

plt.savefig("registration_benchmark.jpg", dpi = 1000)
plt.show()