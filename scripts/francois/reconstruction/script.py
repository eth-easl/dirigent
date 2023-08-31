import matplotlib.pyplot as plt
import numpy as np

# data from https://allisonhorst.github.io/palmerpenguins/

species = (
    "1 service",
    "10 services",
    "100 services",
    "1000 services",
)
weight_counts = {
    "Data plane reconstruction": np.array([2,3,3,13]),
    "Worker reconstruction": np.array([4,5,5,11.5]),
    "Services reconstruction": np.array([8,17,51,397]),
}
width = 0.5

fig, ax = plt.subplots()
bottom = np.zeros(4)

for boolean, weight_count in weight_counts.items():
    p = ax.bar(species, weight_count, width, label=boolean, bottom=bottom)
    bottom += weight_count

ax.set_title("Reconstruction time in milliseconds without endpoints")
ax.legend(loc="upper left")

plt.savefig("reconstruction_no_endpoints.png")