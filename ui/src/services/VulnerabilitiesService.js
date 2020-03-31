import queryString from 'qs';
import { saveFile } from 'services/DownloadService';
import { cveSortFields } from 'constants/sortFields';
import queryService from 'modules/queryService';
import axios from './instance';

const baseUrl = '/v1/cves';
const csvUrl = '/api/vm/export/csv';

/**
 * Send request to suppress / unsuppress CVE with a given ID.
 *
 * @param {!string} CVE unique identifier
 * @param {!boolean} true if CVE should be suppressed, false for unsuppress
 * @returns {Promise<AxiosResponse, Error>} fulfilled in case of success or rejected with an error
 */
export function updateCveSuppressedState(cveIdsToToggle, suppressed = false) {
    return axios.patch(`${baseUrl}/suppress`, { ids: cveIdsToToggle, suppressed });
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
                sortOption
            }
        },
        { arrayFormat: 'repeat', allowDots: true }
    );

    const url = params ? `${csvUrl}?${params}` : csvUrl;

    return saveFile({
        method: 'get',
        url,
        data: null,
        name: `${fileName}.csv`
    });
}

export function exportCvesAsCsv(fileName, workflowState) {
    const fullEntityContext = workflowState.getEntityContext();
    const lastEntityCtx = Object.keys(fullEntityContext).reduce((acc, key) => {
        return { ...{ [key]: fullEntityContext[key] } };
    }, {});

    const query = queryService.objectToWhereClause({
        ...workflowState.getCurrentSearchState(),
        ...queryService.entityContextToQueryObject(lastEntityCtx)
    });

    let sortOption = workflowState.getCurrentSortState()[0];
    sortOption = sortOption && { field: sortOption.id, reversed: sortOption.desc };

    return getCvesInCsvFormat(fileName, query, sortOption);
}
