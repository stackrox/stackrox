import React, { useState } from 'react';
import pluralize from 'pluralize';

import { filterModes } from 'constants/networkFilterModes';
import useSearchFilteredData from 'hooks/useSearchFilteredData';
import { getNetworkFlows } from 'utils/networkUtils/getNetworkFlows';

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

function getPanelHeaderText(numNetworkFlows, filterState) {
    switch (filterState) {
        case filterModes.active:
            return `${numNetworkFlows} active ${pluralize('flow', numNetworkFlows)}`;
        case filterModes.allowed:
            return `${numNetworkFlows} allowed ${pluralize('flow', numNetworkFlows)}`;
        default:
            return `${numNetworkFlows} ${pluralize('flow', numNetworkFlows)}`;
    }
}

function NetworkFlows({ deploymentId, edges, filterState, onNavigateToDeploymentById }) {
    const { networkFlows } = getNetworkFlows(edges, filterState);
    const { data: networkBaselines, isLoading } = useFetchNetworkBaselines({
        deploymentId,
        networkFlows,
        filterState,
    });

    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);

    const filteredNetworkBaselines = useSearchFilteredData(
        networkBaselines,
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
    const panelHeader = getPanelHeaderText(networkFlows.length, filterState);
    const headerComponents = (
        <TablePagination
            page={page}
            dataLength={filteredNetworkBaselines.length}
            setPage={setPage}
        />
    );

    return (
        <Panel id={panelId} header={panelHeader} headerComponents={headerComponents}>
            <div className="p-2 border-b border-base-300">
                <NetworkFlowsSearch
                    networkBaselines={networkBaselines}
                    searchOptions={searchOptions}
                    setSearchOptions={setSearchOptions}
                />
            </div>
            <NetworkFlowsTable
                networkFlows={filteredNetworkBaselines}
                page={page}
                filterState={filterState}
                onNavigateToDeploymentById={onNavigateToDeploymentById}
            />
        </Panel>
    );
}

export default NetworkFlows;
