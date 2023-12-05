#!/usr/bin/env python3

import os
import json
from datetime import datetime, timedelta
import logging
from urllib import request, parse
import time

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s: %(levelname)s: %(message)s")

BASE_URL = os.getenv('SCANNER_NVD_URL', "https://services.nvd.nist.gov/rest/json/cves/2.0")
API_KEY = os.getenv('SCANNER_NVD_API_KEY', None)


def get_dates(year, interval):
    # interval should be a factor of 12
    if 12 % interval != 0:
        raise ValueError("Interval must be a factor of 12.")

    for month in range(1, 13, interval):
        start = datetime(year, month, 1)

        # Handling for months greater than 12 (for intervals like 4, 6 months, etc.)
        end_month = month + interval
        end_year = year
        if end_month > 12:
            end_month -= 12
            end_year += 1

        end = datetime(end_year, end_month, 1) - timedelta(days=1)
        yield start.strftime("%Y-%m-%dT00:00:00.000"), end.strftime("%Y-%m-%dT23:59:59.999")


def fetch_data(start, end):
    index, total = 0, 0
    logging.info(f"Fetching page from {start} to {end}")
    backoff_time = 1
    max_retries = 3

    while total == 0 or index < total:
        try:
            params = {
                'noRejected': '',
                'startIndex': index,
                'pubStartDate': start,
                'pubEndDate': end
            }
            encoded_params = parse.urlencode(params)
            url = f"{BASE_URL}?{encoded_params}"
            req = request.Request(url)
            req.headers = {'apiKey': API_KEY}

            with request.urlopen(req, timeout=45) as response:
                data = json.loads(response.read().decode())
            total = data['totalResults']
            logging.info(f"Fetched page at index {index} (out of {total} total items)")
            yield from data['vulnerabilities']
            index += data['resultsPerPage']
            max_retries = 3 #reset
            backoff_time = 1
        except Exception as e:
            logging.error(f"Failed to download page at index {index}: {e}")
            if max_retries > 0:
                logging.info(f"Retrying in {backoff_time} seconds...")
                time.sleep(backoff_time)
                backoff_time *= 2  # Increase the backoff time exponentially
                max_retries -= 1
            else:
                raise  # Re-raise the exception if max retries have been exceeded
        time.sleep(2)

def main():
    import argparse
    parser = argparse.ArgumentParser()
    if API_KEY is None:
        raise ValueError("API_KEY is not set. Please provide a valid API key.")

    def dirpath(s):
        if os.path.isdir(s):
            return s
        raise ValueError

    parser.add_argument(
        'dirpath',
        help="Path to directory where the NVD data will be saved.",
        type=dirpath)

    args = parser.parse_args()

    try:
        for year in range(2002, datetime.now().year + 1):
            logging.info(f"Fetching year {year}")
            vulnerabilities = []

            for start, end in get_dates(year, 3):# fetch quarterly
                vulnerabilities.extend(fetch_data(start, end))

            file_path = os.path.join(args.dirpath, f"{year}.json")
            with open(file_path, "w") as file:
                json.dump({"vulnerabilities": vulnerabilities}, file)

            file_size = os.path.getsize(file_path) # Get the size of the file
            logging.info(f"Size of {file_path}: {file_size} bytes")
    except Exception as e:
        logging.error(f"An error occurred: {e}")
        raise  # Re-raise the exception

if __name__ == "__main__":
    main()
