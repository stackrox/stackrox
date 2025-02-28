#! /usr/bin/env nix-shell
#! nix-shell -i python3 -p python3 python3Packages.matplotlib python3Packages.pandas python3Packages.seaborn

import json
import pandas
import matplotlib.pyplot as plt
import seaborn
import re
from itertools import chain
import sys

TIMING_DATA_PATTERN = re.compile(r"INFO: .*: .*TIMING_DATA: (.*)")

def extract_timing(line):
    if match := TIMING_DATA_PATTERN.match(line):
        return [json.loads(match.group(1))]
    return []

def load_data(input_stream):
    return pandas.DataFrame(list(chain.from_iterable(extract_timing(line) for line in input_stream)))

def plot(df):
    df["test_step"] = df["test"] + ":" + df["step"]
    df["minutes_spent"] = df["seconds_spent"] / 60.0
    plt.figure(figsize=(30, 20))
    seaborn.barplot(x="test_step", y="minutes_spent", data=df, palette="viridis")
    plt.xticks(rotation=45, ha="right")
    plt.xlabel("Test Step")
    plt.ylabel("Time Spent (Minutes)")
    plt.title("Test Step Execution Times")
    plt.tight_layout()
    plt.savefig(sys.stdout.buffer, format="png")

plot(load_data(sys.stdin))
