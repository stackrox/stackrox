# 0004 - Scanner V4 NVD CVSS enrichment data updating pipeline

- **Author(s):** Yi Li <yli3@redhat.com>, J. Victor Martins <jvdm@sdf.org>
- **Created:** [2023-09-11]

## Context

We are currently transitioning the StackRox Scanner to use ClairCore as its primary scanning engine. Our goals are twofold:

1. Enable Central to retrieve NVD CVSS data and consolidate all CVSS data that contains available CVSS v3 scores.
2. Ensure seamless integration between Scanner V4 and Central, for enriching vulnerabilities with CVSS data.

Throughout this workflow, a new CVSS Updater equipped by a novel enricher in Central will download and consolidate the CVSS data, then creating a JSON bundle. 

ClairCore will be instrumental in scanner V4 data retriever (will be explained in detail below) while requesting this CVSS data bundle from central, handling tasks such as downloading, parsing, and updating CVSS data for Scanner V4.

## Decision

The CVSS Updater will be housed within Central, where it will retrieve CVSS data from Google storage when needed. It is designed to make a single HTTP call for each yearly NVD data bundle, with the earliest data tracing back to 2002. The updater will also compare the sha256 from the CVSS meta file for each year to determine if any data can be bypassed.

This Updater will consolidate the data that only contains valid CVSS v3 scores, generate a json file and store it as a zip file in Central's file system. This compressed file is typically around 50 MB in size.

Using a new CVSS enricher, the updater operates as a GoRoutine, set to refresh the CVSS data bundle every 4 hours. By leveraging GoRoutine, this pipeline runs independently from Central, ensuring that any failures won't disrupt Central.

The CVSS Updater handler, paired with a singleton in Central, offers an HTTP handler for both the sensor and scanner, facilitating data retrieval requests, fetching the existing data bundle and delivering consolidated NVD CVSS data via a http URL.

Within Scanner V4, there's a component named as 'data retriever', which in Scanner V2 was called the 'updater'. This name, 'updater', could be mistaken for the data updating GitHub action workflow in the Scanner V4 context. This CVSS data retriever in Scanner V4 communicates directly with Central's CVSS data retrieval endpoint.

### Central CVSS data retrieving Endpoints

1. `/api/extensions/scanner-v4/definitions`: is configured to handle GET requests, which are processed by the CVSS Updater handler in central.

## Consequences

This document focuses exclusively on the pipeline for updating CVSS enrichment data. 

We are not making any decisions for the updating pipeline of other data (vulneralbility data and repo to cpe mapping data) at this moment.

A novel CVSS enricher mentioned above will be introduced and it will be equipped with an NVD CVSS JSON data parsing feature for the CVSS updater's use. 

Unlike the Claircore CVSS enricher (which can also be utilized as a library/tool in Central), which loads JSON into memory before parsing, this new enricher parses JSON as a stream. The data processing and consolidation managed by this novel enricher utilize approximately 130MB of memory.
