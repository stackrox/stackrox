# 0004 - Scanner V4 NVD CVSS enrichment data updating pipeline

- **Author(s):** Yi Li
- **Created:** [2023-09-11]

## Context

We are in the process of transitioning the StackRox Scanner to utilize ClairCore as its primary scanning engine. Our objective is twofold: firstly, to allow Central to fetch NVD CVSS data, and secondly, to ensure the Scanner V4 seamlessly connecting with Central, triggering data updates and enriching vulnerabilities with CVSS data. Throughout this pipeline, the Claircore enricher will play a pivotal role. It will manage tasks including downloading CVSS data from Google storage, parsing the information, and updating the data accordingly for matcher

## Decision

ScannerV4 APIs will exclusively use gRPC. ScannerV4 APIs are not backward compatible with ScannerV2.

The CVSS Updater resides within Central, fetching CVSS data from Google storage upon request. This updater then stores the data in Central's file system. Throughout this download and save process, the Claircore CVSS enricher plays a significant role.

CVSS Updater handler and singleton in central provides http URL for sensor to connect with and send the data retrieving request and serves with NVD data.

In Scanner V4, the component known as the 'data retriever' (previously referred to as the 'updater' in Scanner V2). However, its name 'updater' might be confused with the 'updater' that operates as a GitHub Action workflow in the Scanner V4 context. This data retriever in Scanner V4 communicates with the CVSS data retrieval endpoint, initiating data downloads in central and ensuring access to the latest CVSS data."

### Central CVSS data retrieving Endpoints

1. `/api/extensions/scannerdefinitions`: is configured to handle GET requests, which are processed by the CVSS Updater handler in central.

## Consequences

This document focuses exclusively on the pipeline for updating CVSS enrichment data. However, the design and structure of this pipeline could also be relevant for the 'repo to cpe' data and 'vulnerabilities' data pipelines. Thus, the architecture and principles underlying the CVSS data update process may potentially be generalized and adopted for all three scenarios in future iterations