# 0004 - Scanner V4 NVD CVSS enrichment data updating pipeline

- **Author(s):** Yi Li
- **Created:** [2023-09-11]

## Context

We are currently transitioning the StackRox Scanner to use ClairCore as its primary scanning engine. Our goals are twofold:

1. Enable Central to retrieve NVD CVSS data and consolidate all CVSS data with available CVSS v3 scores.
2. Ensure seamless integration between Scanner V4 and Central, further enhancing vulnerabilities with CVSS data.

Throughout this workflow, a new CVSS enricher in Central will download and consolidate the CVSS data, then creating a JSON bundle. The ClairCore enricher will be instrumental in this process, handling tasks such as downloading, parsing, and updating CVSS data for Scanner V4.

## Decision

The CVSS Updater resides within Central, fetching CVSS data from Google storage upon request. This updater then stores the data in Central's file system. Throughout this download and save process, the Claircore CVSS enricher plays a significant role.

CVSS Updater handler and singleton in central provides http URL for sensor and scanner to connect with and send the data retrieving request. Most importantly it serves with NVD data upon those requests.

In Scanner V4, the component known as the 'data retriever' (previously referred to as the 'updater' in Scanner V2). However, its name 'updater' might be confused with the 'updater' that operates as a GitHub Action workflow in the Scanner V4 context. This data retriever in Scanner V4 communicates with the CVSS data retrieval endpoint, initiating data downloads in central and ensuring access to the latest CVSS data.

### Central CVSS data retrieving Endpoints

1. `/api/extensions/scannerdefinitions`: is configured to handle GET requests, which are processed by the CVSS Updater handler in central.

## Consequences

This document focuses exclusively on the pipeline for updating CVSS enrichment data. However, the design and structure of this pipeline could also be relevant for the 'repo to cpe' data and 'vulnerabilities' data pipelines. Thus, the architecture and principles underlying the CVSS data update process may potentially be generalized and adopted for all three scenarios in future iterations
