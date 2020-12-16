import React, { useState } from 'react';
import pluralize from 'pluralize';

import { getNetworkFlows } from 'utils/networkGraphUtils';
import { filterModes } from 'constants/networkFilterModes';
import useSearchFilteredData from 'hooks/useSearchFilteredData';
import baselineStatusesData from 'mockData/baselineStatuses';

import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import NetworkFlowsSearch, { getNetworkFlowValueByCategory } from './NetworkFlowsSearch';
import NetworkFlowsTable from './NetworkFlowsTable';

function getPanelId(filterState) {
    switch (filterState) {
        case filterModes.active:
            return 'active-network-flows';
        case filterModes.allowed:
            return 'allowed-network-flows';
        default:
            return 'network-flows';
    }
}

function getPanelHeaderText(networkFlows, filterState) {
    switch (filterState) {
        case filterModes.active:
            return `${networkFlows.length} active ${pluralize('flow', networkFlows.length)}`;
        case filterModes.allowed:
            return `${networkFlows.length} allowed ${pluralize('flow', networkFlows.length)}`;
        default:
            return `${networkFlows.length} ${pluralize('flow', networkFlows.length)}`;
    }
}

function useFetchBaselineStatuses(edges, filterState) {
    // get the network flows for the edges
    // eslint-disable-next-line no-unused-vars
    const { networkFlows } = getNetworkFlows(edges, filterState);
    // TODO: Do the API call to get network flows with baseline statuses
    // return result
    return baselineStatusesData;
}

function NetworkFlows({ edges, filterState, onNavigateToDeploymentById }) {
    const networkFlows = useFetchBaselineStatuses(edges, filterState);
    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);

    const filteredNetworkFlows = useSearchFilteredData(
        networkFlows,
        searchOptions,
        getNetworkFlowValueByCategory
    );

    const panelId = getPanelId(filterState);
    const panelHeader = getPanelHeaderText(filteredNetworkFlows, filterState);
    const headerComponents = (
        <TablePagination page={page} dataLength={filteredNetworkFlows.length} setPage={setPage} />
    );

    return (
        <Panel id={panelId} header={panelHeader} headerComponents={headerComponents}>
            <div className="p-2 border-b border-base-300">
                <NetworkFlowsSearch
                    networkFlows={networkFlows}
                    searchOptions={searchOptions}
                    setSearchOptions={setSearchOptions}
                />
            </div>
            <NetworkFlowsTable
                networkFlows={filteredNetworkFlows}
                page={page}
                filterState={filterState}
                onNavigateToDeploymentById={onNavigateToDeploymentById}
            />
        </Panel>
    );
}

export default NetworkFlows;
