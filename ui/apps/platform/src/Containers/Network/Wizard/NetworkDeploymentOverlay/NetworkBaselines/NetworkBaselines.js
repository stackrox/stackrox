import React, { useState } from 'react';

import { filterModes } from 'constants/networkFilterModes';
import useSearchFilteredData from 'hooks/useSearchFilteredData';

import Panel from 'Components/Panel';
import TablePagination from 'Components/TablePagination';
import Loader from 'Components/Loader';
import NetworkBaselinesSearch, {
    getNetworkBaselineValueByCategory,
} from './NetworkBaselinesSearch';
import NetworkBaselinesTable from './NetworkBaselinesTable';

import useToggleBaselineStatuses from './useToggleBaselineStatuses';

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

function NetworkBaselines({
    header,
    isLoading,
    networkBaselines,
    deploymentId,
    filterState,
    onNavigateToEntity,
}) {
    const toggleBaselineStatuses = useToggleBaselineStatuses(deploymentId);

    const [page, setPage] = useState(0);
    const [searchOptions, setSearchOptions] = useState([]);

    const filteredNetworkBaselines = useSearchFilteredData(
        networkBaselines,
        searchOptions,
        getNetworkBaselineValueByCategory
    );

    if (isLoading) {
        return (
            <div className="p-4 w-full">
                <Loader message={null} />
            </div>
        );
    }

    const panelId = getPanelId(filterState);
    const headerComponents = (
        <TablePagination
            page={page}
            dataLength={filteredNetworkBaselines.length}
            setPage={setPage}
        />
    );

    return (
        <Panel
            id={panelId}
            header={header}
            headerComponents={headerComponents}
            bodyClassName="flex flex-1 flex-col"
        >
            <div className="p-2 border-b border-base-300">
                <NetworkBaselinesSearch
                    networkBaselines={networkBaselines}
                    searchOptions={searchOptions}
                    setSearchOptions={setSearchOptions}
                />
            </div>
            <NetworkBaselinesTable
                networkBaselines={filteredNetworkBaselines}
                page={page}
                filterState={filterState}
                onNavigateToEntity={onNavigateToEntity}
                toggleBaselineStatuses={toggleBaselineStatuses}
            />
        </Panel>
    );
}

export default NetworkBaselines;
