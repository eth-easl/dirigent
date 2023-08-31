import matplotlib.pyplot as plt
import numpy as np

# data from https://allisonhorst.github.io/palmerpenguins/

species = (
    "1 Endpoint",
    "10 Endpoints",
    "100 Endpoints",
    "1000 Endpoints",
)
weight_counts = {
    "Data plane reconstruction": np.array([2,3,4,15]),
    "Worker reconstruction": np.array([4,5,6,16]),
    "Services reconstruction": np.array([51,52,54,57]),
    "Endpoints reconstruction": np.array([2,12,49,334]),
}
width = 0.5

fig, ax = plt.subplots()
bottom = np.zeros(4)

for boolean, weight_count in weight_counts.items():
    p = ax.bar(species, weight_count, width, label=boolean, bottom=bottom)
    bottom += weight_count

ax.set_title("Reconstruction time in milliseconds without endpoints")
ax.legend(loc="upper left")

plt.savefig("reconstruction_endpoints.png")