# example from : https://matplotlib.org/stable/gallery/lines_bars_and_markers/barchart.html#sphx-glr-gallery-lines-bars-and-markers-barchart-py
import matplotlib.pyplot as plt
import numpy as np

array = [""]
values = {
    'Empty - first start': (0.0006),
    '0 Services': (0.002),
    '100 Services': (0.014),
    '1000 Services': (0.080),
    '10000 Services': (0.648),
    '100000 Services': (6.830),
}

x = np.arange(len(array))  # the label locations
width = 0.25  # the width of the bars
multiplier = 0

fig, ax = plt.subplots(layout='constrained')

for attribute, measurement in values.items():
    offset = width * multiplier
    rects = ax.bar(x + offset, measurement, width, label=attribute)
    ax.bar_label(rects, padding=3)
    multiplier += 1

ax.set_ylabel('Latency [s]')
ax.set_title('Time to reconstruct [5 workers - 1 dataplane - x services]')
ax.set_xticks(x + width, array)
ax.legend(loc='upper left', ncols=1)
ax.set_ylim(0, 10)

plt.savefig(f"time_reconstruction.png")

