import { saveFile } from 'services/DownloadService';
import entityTypes from 'constants/entityTypes';
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

export function getCvesInCsvFormat(fileName, searchParamsList) {
    const searchString = searchParamsList
        .map(item => {
            return `${item.key}=${item.value}`;
        })
        .join('&');

    const url = searchString ? `${csvUrl}?${searchString}` : csvUrl;

    return saveFile({
        method: 'get',
        url,
        data: null,
        name: `${fileName}.csv`
    });
}

const searchFields = {
    [entityTypes.CLUSTER]: 'Cluster+ID',
    [entityTypes.COMPONENT]: 'Component+ID',
    [entityTypes.DEPLOYMENT]: 'Deployment+ID',
    [entityTypes.NAMESPACE]: 'Namespace+ID',
    [entityTypes.IMAGE]: 'Image+Sha'
};

export function exportCvesAsCsv(fileName, workflowState) {
    const searchParamsList = [];
    const pageStack = workflowState.getPageStack();
    const ultimateEntity = workflowState.getSkimmedStack();
    const lastItemList = ultimateEntity.getPageStack();

    if (pageStack.length > 1 && pageStack[pageStack.length - 1].t === entityTypes.CVE) {
        // state is on the CVE tab of an entity, so get the parent entity to pass to the CSV endpoint
        const parentEntity = pageStack[pageStack.length - 2];
        const searchEntity = parentEntity.t;
        const id = parentEntity.i;
        if (id) {
            searchParamsList.push({ key: searchFields[searchEntity], value: id });
        }
    } else if (lastItemList[0].t !== entityTypes.CVE) {
        // state is entity sidepanel's CVE list, so get the parent of that CVE list
        const sidebarEntity = lastItemList[0];
        const searchEntity = sidebarEntity.t;
        const id = sidebarEntity.i;
        if (id) {
            searchParamsList.push({ key: searchFields[searchEntity], value: id });
        }
    }
    // if neither of those cases is true, we are on the main CVE list page, and no entity param is needed

    // add Fixable search param, if present
    const searchParams = workflowState.getCurrentSearchState();
    const fixableFlag = Object.keys(searchParams).find(
        key => key.includes('Fix') || key.includes('fix')
    );
    if (fixableFlag) {
        searchParamsList.push({ key: 'Fixable', value: true });
    }

    return getCvesInCsvFormat(fileName, searchParamsList);
}
