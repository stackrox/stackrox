#!/usr/bin/env python3

import argparse
import copy
import json
import sys
from argparse import RawTextHelpFormatter
from statistics import mean


def get_file_paths(file_prefix, num_files):
    """
    Generates the json file paths of the format <file_prefix>idx.json
    where idx is an integer between 0 and num_files-1

    :param file_prefix: The first part of the file path
    that all files to be averaged have in common
    :param num_files: The number of files to be averaged

    :return: A list of files to be averaged
    """
    files = []

    for i in range(num_files):
        files.append(f"{file_prefix}{i}.json")

    return files


def get_metrics_collections(file_paths):
    """
    Reads the json files to be averaged and returns the json contents

    :param file_paths: A list of file paths

    :return: A list of dictionaries, which represent the metrics obtained
    from one run of Collector
    """
    metrics_collections = []

    for file_path in file_paths:
        with open(file_path) as f:
            metrics_collections.append(json.load(f))

    return metrics_collections


def initialize_cumulative(metrics, cumulative):
    """
    Loops through cumulative and sets the value of any entry with the key
    'value' to an empty list

    :param cumulative: A dictionary which will contain results from all runs
    of the performance tests for a particular version of Collector later in
    the program. Here it will only contain empty lists.
    """
    for metric_group in metrics:
        cumulative[metric_group] = {}
        for metric in metrics[metric_group]:
            cumulative[metric_group][metric] = {}
            for key in metrics[metric_group][metric]:
                value = metrics[metric_group][metric][key]
                if type(value) is float or value is None:
                    cumulative[metric_group][metric][key] = []
                else:
                    cumulative[metric_group][metric][key] = value


def accumulate_metric_values_once(metrics, cumulative):
    """
    Loops through the dictionary metrics and appends any values it find to
    corresponding lists in the dictionary cumulative

    :param metrics: A dictionary representing the results of one run of StackRox
    :param cumulative: A dictionary which will contain results from all runs
    of the performance tests for a particular version of Collector
    """
    for metric_group in metrics:
        for metric in metrics[metric_group]:
            for key in metrics[metric_group][metric]:
                value = metrics[metric_group][metric][key]
                if type(value) is float:
                    cumulative[metric_group][metric][key].append(value)


def accumulate_metric_values(metrics_collections, cumulative):
    """
    Loops over a list of dictionaries with the same structure and creates a
    dictionary where each element in the dictionary with the key value is a
    list of the values found in the original list of dictionaries.

    :param metrics_collections: A list of dictionaries representing the results from
    multiple runs of Collector and StackRox
    :param cumulative: A dictionary which will contain results from all runs
    of the performance tests for a particular version of Collector
    """
    for metrics in metrics_collections:
        accumulate_metric_values_once(metrics, cumulative)


def calc_averages_from_cumulative(cumulative):
    """
    Computes the averages of the metrics

    :param cumulative: A dictionary where the values of the metrics are replaced by
    lists representing the different values from different runs of Collector.

    :return: A dictionary with the average results of the runs of Collector.
    """
    averages = {}
    for metric_group in cumulative:
        averages[metric_group] = {}
        for metric in cumulative[metric_group]:
            averages[metric_group][metric] = {}
            for key in cumulative[metric_group][metric]:
                value = cumulative[metric_group][metric][key]
                if type(value) is list:
                    if len(value) > 0:
                        averages[metric_group][metric][key] = mean(value)
                    else:
                        averages[metric_group][metric][key] = None
                else:
                    averages[metric_group][metric][key] = value

    return averages


def calc_averages(file_prefix, num_files):
    """
    Computes the average results over multiple runs

    :param file_prefix: The first part of the file path
    that all files to be averaged have in common
    :param num_files: The number of files to be averaged

    :return: A dictionary with the average of the results of the different runs
    """

    file_paths = get_file_paths(file_prefix, num_files)

    # metrics_collections is a list of dictionaries, each of which represents the metrics
    # obtained from one run of Collector
    metrics_collections = get_metrics_collections(file_paths)

    cumulative = {}
    initialize_cumulative(metrics_collections[0], cumulative)
    accumulate_metric_values(metrics_collections, cumulative)
    averages = calc_averages_from_cumulative(cumulative)

    return averages


if __name__ == '__main__':
    description = ('Calculates the average of metrics '
                   'from multiple runs of StackRox')
    parser = argparse.ArgumentParser(
                                    description=description,
                                    formatter_class=RawTextHelpFormatter
                                    )

    file_prefix_help = \
        """The json file paths to be averaged over, not including
index and extension.
The following is an example of what an input file might look like:
{
    "collector_timers": {
        "net_scrape_update": {
            "description": "Time taken by net_scrape_update",
            "units": "microseconds",
            "Average": 718.1666666666667,
            "Maximum": 1822.0
        },
        "net_scrape_read": {
            "description": "Time taken by net_scrape_read",
            "units": "microseconds",
            "Average": 100571.83333333333,
            "Maximum": 140619.0
        }
    }
}
"""

    num_files_help = 'The number of files to average over'

    parser.add_argument('filePrefix', help=file_prefix_help)
    parser.add_argument('numFiles', type=int, help=num_files_help)
    parser.add_argument('outputFile', help='Where the output is written to')
    args = parser.parse_args()

    file_prefix = args.filePrefix
    num_files = args.numFiles
    output_file = args.outputFile

    averages = calc_averages(file_prefix, num_files)
    with open(output_file, 'w') as json_file:
        json.dump(averages, json_file, indent=4, separators=(',', ': '))
