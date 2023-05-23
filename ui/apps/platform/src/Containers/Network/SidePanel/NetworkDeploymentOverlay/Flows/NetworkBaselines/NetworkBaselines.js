import React, { useState } from 'react';

import { filterModes } from 'constants/networkFilterModes';
import useSearchFilteredData from 'hooks/useSearchFilteredData';

import { PanelNew, PanelBody, PanelHead, PanelHeadEnd, PanelTitle } from 'Components/Panel';
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
    headerComponents,
    isLoading,
    networkBaselines,
    deploymentId,
    filterState,
    onNavigateToEntity,
    includedBaselineStatuses,
    excludedSearchCategories,
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

    return (
        <PanelNew testid={panelId}>
            <PanelHead>
                <PanelTitle testid={`${panelId}-header`} text={header} />
                <PanelHeadEnd>
                    {headerComponents}
                    <TablePagination
                        page={page}
                        dataLength={filteredNetworkBaselines.length}
                        setPage={setPage}
                    />
                </PanelHeadEnd>
            </PanelHead>
            <PanelHead>
                <PanelHeadEnd>
                    <div className="pr-3 w-full">
                        <NetworkBaselinesSearch
                            networkBaselines={networkBaselines}
                            searchOptions={searchOptions}
                            setSearchOptions={setSearchOptions}
                            excludedSearchCategories={excludedSearchCategories}
                        />
                    </div>
                </PanelHeadEnd>
            </PanelHead>
            <PanelBody>
                <NetworkBaselinesTable
                    networkBaselines={filteredNetworkBaselines}
                    page={page}
                    filterState={filterState}
                    onNavigateToEntity={onNavigateToEntity}
                    toggleBaselineStatuses={toggleBaselineStatuses}
                    includedBaselineStatuses={includedBaselineStatuses}
                    excludedColumns={excludedSearchCategories}
                />
            </PanelBody>
        </PanelNew>
    );
}

export default NetworkBaselines;
