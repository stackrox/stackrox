import json
import os
import sys

from opensearchpy import OpenSearch, RequestsHttpConnection


def read_results(path):
    with open(path) as f:
        data = []
        for row in f:
            parsed_json = json.loads(row)
            if parsed_json["type"] in ["Point"] and parsed_json["metric"] not in ["group_duration", "data_sent", "data_received", "iteration_duration", "iterations", "vus", "vus_max"]:
                data.append(parsed_json)
        return data


def payload_constructor(data, action):
    action_string = json.dumps(action) + "\n"
    payload_string=""

    for datum in data:
        payload_string += action_string
        this_line = json.dumps(datum) + "\n"
        payload_string += this_line
    return payload_string


def main(path):
    host = os.getenv("K6_ELASTICSEARCH_URL")
    port = 443
    auth = (os.getenv("K6_ELASTICSEARCH_USER"), os.getenv("K6_ELASTICSEARCH_PASSWORD"))

    client = OpenSearch(
        hosts=[{'host': host, 'port': port}],
        http_auth=auth,
        use_ssl=True,
        verify_certs=True,
        connection_class=RequestsHttpConnection
    )

    info = client.info()
    print(f"Welcome to {info['version']['distribution']} {info['version']['number']}!")

    # Create indices
    target_index = 'k6-metrics'

    action={
        "index": {
            "_index": target_index
        }
    }

    data = read_results(path)
    response = client.bulk(body=payload_constructor(data, action), index=target_index)
    if response["errors"]:
        print(response)
        exit(1)

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: opensearch-json-import.py <path to raw JSON>")
        exit(1)
    main(sys.argv[1])
