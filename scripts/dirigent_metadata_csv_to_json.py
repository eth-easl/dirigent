import json
import sys

import pandas as pd

path = sys.argv[1]

df = pd.read_csv(f"{path}/dirigent.csv")
dictionary = []


def noParser(val):
    return val


def blankSpaceParser(val):
    return val.split(' ')


def setIfExists(df, entry, row, column, default=None, parser=noParser):
    if column in df.columns:
        entry[column] = parser(row[column])
    else:
        entry[column] = default


for idx, row in df.iterrows():
    entry = {
        "HashFunction": row["HashFunction"],
        "Image": row["Image"],
        "Port": int(row["Port"]),
        "Protocol": row["Protocol"],
        "ScalingUpperBound": row["ScalingUpperBound"],
        "ScalingLowerBound": row["ScalingLowerBound"],
        "IterationMultiplier": row["IterationMultiplier"],
    }

    setIfExists(df, entry, row, "IOPercentage", default=0, parser=noParser)
    setIfExists(df, entry, row, "EnvVars", default=[], parser=blankSpaceParser)
    setIfExists(df, entry, row, "ProgramArgs", default=[], parser=blankSpaceParser)

    dictionary.append(entry)

json_object = json.dumps(dictionary, indent=4)

# Writing to sample.json
with open(f"{path}/dirigent.json", "w") as outfile:
    outfile.write(json_object)
