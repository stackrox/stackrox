import React, { useState } from 'react';
import pluralize from 'pluralize';

import { filterModes } from 'constants/networkFilterModes';
import useSearchFilteredData from 'hooks/useSearchFilteredData';

import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import Loader from 'Components/Loader';
import NetworkFlowsSearch, { getNetworkFlowValueByCategory } from './NetworkFlowsSearch';
import NetworkFlowsTable from './NetworkFlowsTable';

import useFetchNetworkBaselines from './useFetchNetworkBaselines';

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

function NetworkFlows({ deploymentId, edges, filterState, onNavigateToDeploymentById }) {
    const { networkBaselines: networkFlows, isLoading } = useFetchNetworkBaselines({
        deploymentId,
        edges,
        filterState,
    });

    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);

    const filteredNetworkFlows = useSearchFilteredData(
        networkFlows,
        searchOptions,
        getNetworkFlowValueByCategory
    );

    if (isLoading) {
        return (
            <div className="p-4 w-full">
                <Loader message={null} />
            </div>
        );
    }

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
