import queryString from 'qs';
import { saveFile } from 'services/DownloadService';
import { cveSortFields } from 'constants/sortFields';
import queryService from 'utils/queryService';
import entityTypes from 'constants/entityTypes';
import axios from './instance';

function getCSVExportUrl(cveType) {
    if (cveType === entityTypes.CLUSTER_CVE) {
        return '/api/export/csv/cluster/cve';
    }
    if (cveType === entityTypes.NODE_CVE) {
        return '/api/export/csv/node/cve';
    }
    if (cveType === entityTypes.IMAGE_CVE) {
        return '/api/export/csv/image/cve';
    }
    // @TODO: Remove this URL when we remove feature flagging for ROX_VM_FRONTEND_UPDATES
    return '/api/vm/export/csv';
}

function getBaseCveUrl(cveType) {
    if (cveType === entityTypes.CLUSTER_CVE) {
        return '/v1/clustercves';
    }
    if (cveType === entityTypes.NODE_CVE) {
        return '/v1/nodecves';
    }
    if (cveType === entityTypes.IMAGE_CVE) {
        return '/v1/imagecves';
    }
    return '/v1/cves';
}

/**
 * Send request to suppress CVEs with a given IDs.
 *
 * @param {!string} CVE unique identifier
 * @param {!string} CVE suppress duration, if 0 then CVEs are suppressed indefinitely
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function suppressVulns(cveType, cveNames, duration = 0) {
    const baseUrl = getBaseCveUrl(cveType);
    return axios.patch(`${baseUrl}/suppress`, { cves: cveNames, duration });
}

/**
 * Send request to unsuppress CVEs with a given IDs.
 *
 * @param {!string} CVE unique identifier
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function unsuppressVulns(cveType, cveNames) {
    const baseUrl = getBaseCveUrl(cveType);
    return axios.patch(`${baseUrl}/unsuppress`, { cves: cveNames });
}

export function getCvesInCsvFormat(
    cveType,
    fileName,
    query,
    sortOption = { field: cveSortFields.CVSS_SCORE, reversed: true },
    page = 0,
    pageSize = 0
) {
    const csvUrl = getCSVExportUrl(cveType);
    const offset = page * pageSize;
    const params = queryString.stringify(
        {
            query,
            pagination: {
                offset,
                limit: pageSize,
                sortOption,
            },
        },
        { arrayFormat: 'repeat', allowDots: true }
    );

    const url = params ? `${csvUrl}?${params}` : csvUrl;

    return saveFile({
        method: 'get',
        url,
        data: null,
        name: `${fileName}.csv`,
    });
}

export function exportCvesAsCsv(fileName, workflowState, cveType) {
    const fullEntityContext = workflowState.getEntityContext();
    const lastEntityCtx = Object.keys(fullEntityContext).reduce((acc, key) => {
        return { ...{ [key]: fullEntityContext[key] } };
    }, {});

    const currentSearchState = workflowState.getCurrentSearchState();

    // TODO: remove after Postgres is required for all installations
    // kludge to make Findings sections CSV export of CVEs work until Postgres is on
    if (cveType === entityTypes.CVE && !currentSearchState['CVE Type']) {
        if (fullEntityContext.NODE) {
            currentSearchState['CVE Type'] = 'NODE_CVE';
        } else {
            currentSearchState['CVE Type'] = 'IMAGE_CVE';
        }
    }

    const query = queryService.objectToWhereClause({
        ...currentSearchState,
        ...queryService.entityContextToQueryObject(lastEntityCtx),
    });

    let sortOption = workflowState.getCurrentSortState()[0];
    sortOption = sortOption && { field: sortOption.id, reversed: sortOption.desc };

    return getCvesInCsvFormat(cveType, fileName, query, sortOption);
}
