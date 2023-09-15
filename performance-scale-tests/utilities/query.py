#!/usr/bin/env python3

import argparse
import json
import requests
import sys

g_stat_names = {'avg': 'Average', 'max': 'Maximum'}
g_components= ['collector', 'sensor', 'central', 'central-db', 'scanner', 'scanner-db', 'admission-control']


class Querier:
    def __init__(self, token, url):
        self.token = token
        self.url = url

    def query(self, query):
        """
        Performs a Prometheus query

        :param query: A Prometheus query

        :return: The response to the Prometheus query in json format
        """
        headers = {
            'Authorization': f'Bearer {self.token}',
            'accept': 'application/json'
        }

        params = {"query": query}

        url = self.url
        resp = requests.get(url, headers=headers, params=params, verify=False)

        resp.raise_for_status()

        return resp.json()

    def query_and_get_value(self, query):
        """
        Perform a Prometheus query and return the value from it
        in the following format {value: 'value'}

        :param query: A Prometheus query

        :return: The value returned by the Prometheus query
        in the following format {value: 'value'}
        """
        response = self.query(query)

        try:
            # Example response:
            # {
            #   "status": "success",
            #   "data": {
            #     "resultType": "vector",
            #     "result": [
            #       {
            #         "metric": {
            #           "job": "kubelet"
            #         },
            #         "value": [
            #           1650322259.04,
            #           "259264.2228070175"
            #         ]
            #       }
            #     ]
            #   }
            # }

            return float(response['data']['result'][0]['value'][1])
        except (KeyError, ValueError, IndexError):
            return None

    def get_stats_for_query(self, query):
        """
        Gets statics in g_stats_names for a query, by adding {stat} by (job)
        to the begining of the query.

        :param query: The query for which the statics are obtained

        :return: A json with statics
        """
        result = {}
        for stat in g_stat_names:
            stat_query = f'{stat} by (job) {query}'
            result[g_stat_names[stat]] = self.query_and_get_value(stat_query)

        return result


def get_collector_timers(querier):
    """
    Performs Prometheus queries relevant to Collector timers

    :param querier: An object containing information
    needed to query the Prometheus API

    :return: Json containing information about Collector timers
    """
    res = {}
    timers = [
                "net_scrape_update",
                "net_scrape_read",
                "net_write_message",
                "net_create_message"
            ]

    total_time = {}
    for stat in g_stat_names:
        total_time[g_stat_names[stat]] = 0

    for timer in timers:
        res[timer] = {'description': f'Time taken by {timer}', 'units': 'microseconds'}
        for stat in g_stat_names:
            time_query = f'{stat} by (job) (rox_collector_timers{{type="{timer}_times_us_avg"}})'

            time_value = querier.query_and_get_value(time_query)
            res[timer][g_stat_names[stat]] = time_value

            try:
                total_time[g_stat_names[stat]] += time_value
            except TypeError:
                err_msg = "WARNING: The following query did not return a valid response\n"
                sys.stderr.write(f'{err_msg}\n')
                sys.stderr.write(f'{time_query}\n')

    metric_name = 'collector_timers_total'
    res[metric_name] = total_time
    res[metric_name]['description'] = 'Total time taken by collector timers'
    res[metric_name]['units'] = 'microseconds'

    return res


def get_collector_counters(querier):
    """
    Performs Prometheus queries relevant to Collector counters

    :param querier: An object containing information
    needed to query the Prometheus API

    :return: Json containing information about Collector counters
    """
    res = {}
    counters = ["net_conn_deltas", "net_conn_updates", "net_conn_inactive"]
    counters += ["net_cep_deltas", "net_cep_updates", "net_cep_inactive"]

    for counter in counters:
        query = f'(rox_collector_counters{{type="{counter}"}})'
        query_name = f'{counter}'

        res[query_name] = querier.get_stats_for_query(query)
        res[query_name]['units'] = "num"

    return res


def get_sensor_network_flows(querier, query_window):
    """
    Performs Prometheus queries relevant to sensor network flows

    :param querier: An object containing information
    needed to query the Prometheus API
    :param query_window: The window over which rates are computed by Prometheus

    :return: Json containing information about sensor network flows
    """
    res = {}
    for flow_type in ["incoming", "outgoing"]:
        for protocol in ["L4_PROTOCOL_TCP", "L4_PROTOCOL_UDP"]:
            metric = 'rox_sensor_network_flow_total_per_node'
            query = f'({metric}{{Protocol="{protocol}",Type="{flow_type}"}})'
            query_name = f'{flow_type}_{protocol}'

            res[query_name] = querier.get_stats_for_query(query)
            res[query_name]['description'] = f'Number of {flow_type} {protocol} network flows over pods'
            res[query_name]['units'] = "num"

    metrics = [
                "rox_sensor_network_flow_host_connections_added",
                "rox_sensor_network_flow_host_connections_removed",
                "rox_sensor_network_flow_external_flows"
            ]

    for metric in metrics:
        query = metric
        rate_query = f'rate({metric}[{query_window}])'

        query_name = metric
        rate_query_name = f'{metric}_rate'

        res[query_name] = {'value': querier.query_and_get_value(query)}
        res[rate_query_name] = {'value': querier.query_and_get_value(rate_query)}
        res[rate_query_name]['description'] = f'Average {metric}'
        res[rate_query_name]['units'] = 'per seconds'

    return res


def get_cpu_mem_network_usage(querier, query_window):
    """
    Performs Prometheus queries relevant to cpu, memory, and network IO usage

    :param querier: An object containing information
    needed to query the Prometheus API
    :param query_window: The window over which rates are computed by Prometheus

    :return: Json containing information about cpu, memory,
    and network IO usage
    """
    res = {}

    cpu_query = f'(rate(container_cpu_usage_seconds_total{{namespace="stackrox"}}[{query_window}]) * 100)'
    cpu_query_name = 'cpu_usage'
    res[cpu_query_name] = querier.get_stats_for_query(cpu_query)
    res[cpu_query_name]['units'] = "% cpu usage per container"

    mem_query = '(container_memory_usage_bytes{namespace="stackrox"})'
    mem_query_name = 'mem_usage'
    res[mem_query_name] = querier.get_stats_for_query(mem_query)
    res[mem_query_name]['units'] = 'bytes'

    for component in g_components:
        component_name = component.replace('-', '_')
        component_cpu_query = f'(rate(container_cpu_usage_seconds_total{{pod=~"{component}-[0-z]{{4}}.*"}}[{query_window}]) * 100)'
        component_cpu_query_name = f'{component_name}_cpu_usage'
        res[component_cpu_query_name] = querier.get_stats_for_query(component_cpu_query)
        res[component_cpu_query_name]['units'] = "% cpu usage per container"

        component_mem_query = f'(rate(container_memory_usage_bytes{{pod=~"{component}-[0-z]{{4}}.*"}}[{query_window}]))'
        component_mem_query_name = f'{component_name}_mem_usage'
        res[component_mem_query_name] = querier.get_stats_for_query(component_mem_query)
        res[component_mem_query_name]['units'] = 'bytes'


    metric_names = {
            'container_network_receive_bytes_total':
            'network_received',
            'container_network_transmit_bytes_total':
            'network_transmited'
            }

    for name in metric_names:
        query = f'({name}{{namespace="stackrox"}})'
        query_name = metric_names[name]
        res[query_name] = querier.get_stats_for_query(query)
        res[query_name]['description'] = f'Total {metric_names[name]}'
        res[query_name]['units'] = 'bytes'

        rate_query = f'(rate({name}{{namespace="stackrox"}}[{query_window}]))'
        rate_query_name = f'rate_{metric_names[name]}'
        res[rate_query_name] = querier.get_stats_for_query(rate_query)
        res[rate_query_name]['units'] = 'bytes per second'

    return res


def get_pod_restarts(querier):
    """
    Performs Prometheus queries relevant to pod restarts

    :param querier: An object containing information
    needed to query the Prometheus API

    :return: Json containing information about pod restarts
    and network IO usage
    """
    res = {}

    for component in g_components:
        component_name = component.replace('-', '_')
        component_restarts_query = f'(kube_pod_container_status_restarts_total{{pod=~"{component}-[0-z]{{4}}.*"}})'
        component_restarts_query_name = f'{component_name}_restarts'
        res[component_restarts_query_name] = querier.get_stats_for_query(component_restarts_query)
        res[component_restarts_query_name]['units'] = 'num'

    return res


def get_central_metrics(querier):
    """
    Performs Prometheus queries relevant to processes listening on ports in central

    :param querier: An object containing information
    needed to query the Prometheus API

    :return: Json containing information about processes listening on ports in central
    """
    res = {}

    for operation in ['Remove', 'RemoveMany', 'UpdateMany']:
        query = f'rox_central_postgres_op_duration_sum{{Operation="{operation}",Type="ProcessListeningOnPortStorage"}}'
        query_name = f'postgres_op_duration_ProcessListeningOnPortStorage_{operation}'
        res[query_name] = {'value': querier.query_and_get_value(query)}
        res[query_name]['units'] = 'ms'

    query = f'rox_central_datastore_function_duration_sum{{Function="AddProcessListeningOnPort",Type="ProcessListeningOnPort"}}'
    query_name = f'datastore_function_duration_sum_AddProcessListeningOnPort'
    res[query_name] = {'value': querier.query_and_get_value(query)}
    res[query_name]['units'] = 'ms'

    query = f'rox_central_function_segment_duration_sum{{Segment="FlushingIndicatorQueue"}} / rox_central_function_segment_duration_count{{Segment="FlushingIndicatorQueue"}}'
    query_name = 'rox_central_function_segment_duration_FlushingIndicatorQueue'
    res[query_name] = {'value': querier.query_and_get_value(query)}
    res[query_name]['units'] = 'ms'

    return res

def main(query_window, token, url_base, output_file):
    """
    Performs Prometheus queries related to the performance of Collector
    and StackRox

    :param query_window: The window over which rates are computed by Prometheus
    :param token: The token used to authenticate for the Prometheus API
    :param url: The Prometheus API endpoint
    :output_file: Where the results are written to
    """

    url=f'https://{url_base}/api/v1/query'
    querier = Querier(token, url)

    collector_timers = get_collector_timers(querier)
    collector_counters = get_collector_counters(querier)
    sensor_network_flows = get_sensor_network_flows(querier, query_window)
    cpu_mem_network_usage = get_cpu_mem_network_usage(querier, query_window)
    pod_restarts = get_pod_restarts(querier)
    central_metrics = get_central_metrics(querier)

    json_output = {
                    'collector_timers': collector_timers,
                    'collector_counters': collector_counters,
                    'sensor_network_flows': sensor_network_flows,
                    'cpu_mem_network_usage': cpu_mem_network_usage,
                    'pod_restarts': pod_restarts,
                    'central_metrics': central_metrics
    }

    with open(output_file, 'w') as json_file:
        json.dump(json_output, json_file, indent=4, separators=(',', ': '))


if __name__ == '__main__':
    description = ('Performs Prometheus queries '
                   'to get performance metrics on StackRox and Collector')

    query_window_help = ('The window of time over '
                         'which Prometheus looks at the data. Eg 10m or 90s')

    token_help = ('The token used for the Prometheus query API. '
                  'You should be able to obtain it by running query.sh')

    url_help = ('The Prometheus API endpoint. '
                'You should be able to obtain it by running query.sh')

    output_file_help = 'The path of the file were the results are written to'

    parser = argparse.ArgumentParser(description=description)
    parser.add_argument('queryWindow', help=query_window_help)
    parser.add_argument('token', help=token_help)
    parser.add_argument('url_base', help=url_help)
    parser.add_argument('outputFile', help=output_file_help)
    args = parser.parse_args()

    main(args.queryWindow, args.token, args.url_base, args.outputFile)
