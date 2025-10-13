import React, { useMemo } from 'react';
import {
    Flex,
    PageSection,
    Pagination,
    pluralize,
    Skeleton,
    Split,
    SplitItem,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import ComponentScannableStatusDropdown from 'Containers/Vulnerabilities/components/ComponentScannableStatusDropdown';
import type { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import { onURLSearch } from 'Components/CompoundSearchFilter/utils/utils';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseUrlSearchReturn } from 'hooks/useURLSearch';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { VirtualMachine } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';
import { getHasSearchApplied } from 'utils/searchUtils';

import {
    applyVirtualMachinePackagesTableFilters,
    applyVirtualMachinePackagesTableSort,
    getVirtualMachinePackagesTableData,
} from '../aggregateUtils';
import { virtualMachineComponentSearchFilterConfig } from '../../searchFilterConfig';
import VirtualMachinePackagesTable from './VirtualMachinePackagesTable';

export type VirtualMachinePagePackagesProps = {
    virtualMachineData: VirtualMachine | undefined;
    isLoadingVirtualMachineData: boolean;
    errorVirtualMachineData: Error | undefined;
    urlSearch: UseUrlSearchReturn;
    urlSorting: UseURLSortResult;
    urlPagination: UseURLPaginationResult;
};

const searchFilterConfig = [virtualMachineComponentSearchFilterConfig];

function VirtualMachinePagePackages({
    virtualMachineData,
    isLoadingVirtualMachineData,
    errorVirtualMachineData,
    urlSearch,
    urlSorting,
    urlPagination,
}: VirtualMachinePagePackagesProps) {
    const { searchFilter, setSearchFilter } = urlSearch;
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const { sortOption, getSortParams } = urlSorting;

    const isFiltered = getHasSearchApplied(searchFilter);

    const virtualMachinePackagesTableData = useMemo(
        () => getVirtualMachinePackagesTableData(virtualMachineData),
        [virtualMachineData]
    );

    const filteredVirtualMachinePackagesTableData = useMemo(
        () =>
            applyVirtualMachinePackagesTableFilters(virtualMachinePackagesTableData, searchFilter),
        [virtualMachinePackagesTableData, searchFilter]
    );

    const sortedVirtualMachinePackagesTableData = useMemo(
        () =>
            applyVirtualMachinePackagesTableSort(
                filteredVirtualMachinePackagesTableData,
                Array.isArray(sortOption) ? sortOption[0].field : sortOption.field,
                Array.isArray(sortOption) ? sortOption[0].reversed : sortOption.reversed
            ),
        [filteredVirtualMachinePackagesTableData, sortOption]
    );

    const paginatedVirtualMachinePackagesTableData = useMemo(() => {
        const totalRows = sortedVirtualMachinePackagesTableData.length;
        const maxPage = Math.max(1, Math.ceil(totalRows / perPage) || 1);
        const safePage = Math.min(page, maxPage);

        const start = (safePage - 1) * perPage;
        const end = start + perPage;
        return sortedVirtualMachinePackagesTableData.slice(start, end);
    }, [sortedVirtualMachinePackagesTableData, page, perPage]);

    const tableState = getTableUIState({
        isLoading: isLoadingVirtualMachineData,
        data: paginatedVirtualMachinePackagesTableData,
        error: errorVirtualMachineData,
        searchFilter,
    });

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    const onSearch = (payload: OnSearchPayload) => {
        onURLSearch(searchFilter, setSearchFilter, payload);
        setPage(1);
    };

    const onScannableStatusSelect = (
        filterType: 'SCANNABLE',
        checked: boolean,
        selection: string
    ) => {
        const action = checked ? 'ADD' : 'REMOVE';
        const category = filterType;
        const value = selection;
        onURLSearch(searchFilter, setSearchFilter, { action, category, value });
        setPage(1);
    };

    return (
        <PageSection variant="light" isFilled padding={{ default: 'padding' }}>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarGroup className="pf-v5-u-w-100">
                        <ToolbarItem className="pf-v5-u-flex-1">
                            <CompoundSearchFilter
                                config={searchFilterConfig}
                                searchFilter={searchFilter}
                                onSearch={onSearch}
                            />
                        </ToolbarItem>
                        <ToolbarItem>
                            <ComponentScannableStatusDropdown
                                searchFilter={searchFilter}
                                onSelect={onScannableStatusSelect}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup className="pf-v5-u-w-100">
                        <SearchFilterChips
                            searchFilter={searchFilter}
                            onFilterChange={setSearchFilter}
                            filterChipGroupDescriptors={[
                                { displayName: 'Scannable Status', searchFilterName: 'SCANNABLE' },
                                { displayName: 'Component', searchFilterName: 'Component' },
                                { displayName: 'Version', searchFilterName: 'Component Version' },
                            ]}
                        />
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>
            <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                    <SplitItem isFilled>
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                            <Title headingLevel="h2">
                                {!isLoadingVirtualMachineData ? (
                                    `${pluralize(filteredVirtualMachinePackagesTableData.length, 'result')} found`
                                ) : (
                                    <Skeleton screenreaderText="Loading virtual machine vulnerability count" />
                                )}
                            </Title>
                            {isFiltered && <DynamicTableLabel />}
                        </Flex>
                    </SplitItem>
                    <SplitItem>
                        <Pagination
                            itemCount={filteredVirtualMachinePackagesTableData.length}
                            perPage={perPage}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </SplitItem>
                </Split>
                <VirtualMachinePackagesTable
                    tableState={tableState}
                    getSortParams={getSortParams}
                    onClearFilters={onClearFilters}
                />
            </div>
        </PageSection>
    );
}

export default VirtualMachinePagePackages;
