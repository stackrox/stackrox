import queryString from 'qs';
import { saveFile } from 'services/DownloadService';
import { cveSortFields } from 'constants/sortFields';
import queryService from 'utils/queryService';
import axios from './instance';

const baseUrl = '/v1/cves';
const csvUrl = '/api/vm/export/csv';

/**
 * Send request to suppress CVEs with a given IDs.
 *
 * @param {!string} CVE unique identifier
 * @param {!string} CVE suppress duration, if 0 then CVEs are suppressed indefinitely
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function suppressVulns(cveIds, duration = 0) {
    return axios.patch(`${baseUrl}/suppress`, { ids: cveIds, duration });
}

/**
 * Send request to unsuppress CVEs with a given IDs.
 *
 * @param {!string} CVE unique identifier
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function unsuppressVulns(cveIds) {
    return axios.patch(`${baseUrl}/unsuppress`, { ids: cveIds });
}

export function getCvesInCsvFormat(
    fileName,
    query,
    sortOption = { field: cveSortFields.CVSS_SCORE, reversed: true },
    page = 0,
    pageSize = 0
) {
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

export function exportCvesAsCsv(fileName, workflowState) {
    const fullEntityContext = workflowState.getEntityContext();
    const lastEntityCtx = Object.keys(fullEntityContext).reduce((acc, key) => {
        return { ...{ [key]: fullEntityContext[key] } };
    }, {});

    const query = queryService.objectToWhereClause({
        ...workflowState.getCurrentSearchState(),
        ...queryService.entityContextToQueryObject(lastEntityCtx),
    });

    let sortOption = workflowState.getCurrentSortState()[0];
    sortOption = sortOption && { field: sortOption.id, reversed: sortOption.desc };

    return getCvesInCsvFormat(fileName, query, sortOption);
}
