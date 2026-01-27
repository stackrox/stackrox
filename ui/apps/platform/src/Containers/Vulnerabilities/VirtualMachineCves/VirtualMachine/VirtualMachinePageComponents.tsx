import { useMemo } from 'react';
import {
    Flex,
    PageSection,
    Pagination,
    Skeleton,
    Split,
    SplitItem,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    pluralize,
} from '@patternfly/react-core';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import type {
    OnSearchCallback,
    SelectSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseUrlSearchReturn } from 'hooks/useURLSearch';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { VirtualMachine } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';
import { getHasSearchApplied } from 'utils/searchUtils';

import {
    applyVirtualMachineComponentsTableFilters,
    applyVirtualMachineComponentsTableSort,
    getVirtualMachineComponentsTableData,
} from '../aggregateUtils';
import { virtualMachineComponentSearchFilterConfig } from '../../searchFilterConfig';
import { scannableStatuses } from '../../types';
import VirtualMachineComponentsPageTable from './VirtualMachineComponentsPageTable';

export const attributeForScannable: SelectSearchFilterAttribute = {
    displayName: 'Scan status',
    filterChipLabel: 'Scan status',
    searchTerm: 'SCANNABLE', // TODO can it become 'Scannable' instead of ALL CAPS before GA?
    inputType: 'select',
    inputProps: {
        // TODO can value become true and file instead of Scanned and Not scanned before GA?
        options: scannableStatuses.map((label) => ({ label, value: label })),
    },
};

export type VirtualMachinePageComponentsProps = {
    virtualMachine: VirtualMachine | undefined;
    isLoadingVirtualMachine: boolean;
    errorVirtualMachine: Error | undefined;
    urlSearch: UseUrlSearchReturn;
    urlSorting: UseURLSortResult;
    urlPagination: UseURLPaginationResult;
};

const searchFilterConfig = [virtualMachineComponentSearchFilterConfig];

function VirtualMachinePageComponents({
    virtualMachine,
    isLoadingVirtualMachine,
    errorVirtualMachine,
    urlSearch,
    urlSorting,
    urlPagination,
}: VirtualMachinePageComponentsProps) {
    const { searchFilter, setSearchFilter } = urlSearch;
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const { sortOption, getSortParams } = urlSorting;

    const isFiltered = getHasSearchApplied(searchFilter);

    const virtualMachineComponentsTableData = useMemo(
        () => getVirtualMachineComponentsTableData(virtualMachine),
        [virtualMachine]
    );

    const filteredVirtualMachineComponentsTableData = useMemo(
        () =>
            applyVirtualMachineComponentsTableFilters(
                virtualMachineComponentsTableData,
                searchFilter
            ),
        [virtualMachineComponentsTableData, searchFilter]
    );

    const sortedVirtualMachineComponentsTableData = useMemo(
        () =>
            applyVirtualMachineComponentsTableSort(
                filteredVirtualMachineComponentsTableData,
                Array.isArray(sortOption) ? sortOption[0].field : sortOption.field,
                Array.isArray(sortOption) ? sortOption[0].reversed : sortOption.reversed
            ),
        [filteredVirtualMachineComponentsTableData, sortOption]
    );

    const paginatedVirtualMachineComponentsTableData = useMemo(() => {
        const totalRows = sortedVirtualMachineComponentsTableData.length;
        const maxPage = Math.max(1, Math.ceil(totalRows / perPage) || 1);
        const safePage = Math.min(page, maxPage);

        const start = (safePage - 1) * perPage;
        const end = start + perPage;
        return sortedVirtualMachineComponentsTableData.slice(start, end);
    }, [sortedVirtualMachineComponentsTableData, page, perPage]);

    const tableState = getTableUIState({
        isLoading: isLoadingVirtualMachine,
        data: paginatedVirtualMachineComponentsTableData,
        error: errorVirtualMachine,
        searchFilter,
    });

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    const onSearch: OnSearchCallback = (payload) => {
        setSearchFilter(updateSearchFilter(searchFilter, payload));
        setPage(1);
    };

    const onSearchScannable: OnSearchCallback = (payload) => {
        setSearchFilter(updateSearchFilter(searchFilter, payload));
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
                            <SearchFilterSelectInclusive
                                attribute={attributeForScannable}
                                isSeparate
                                onSearch={onSearchScannable}
                                searchFilter={searchFilter}
                            />
                        </ToolbarItem>
                    </ToolbarGroup>
                    <ToolbarGroup className="pf-v5-u-w-100">
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={[attributeForScannable]}
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onFilterChange={setSearchFilter}
                        />
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>
            <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                    <SplitItem isFilled>
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                            <Title headingLevel="h2">
                                {!isLoadingVirtualMachine ? (
                                    `${pluralize(filteredVirtualMachineComponentsTableData.length, 'result')} found`
                                ) : (
                                    <Skeleton screenreaderText="Loading virtual machine vulnerability count" />
                                )}
                            </Title>
                            {isFiltered && <DynamicTableLabel />}
                        </Flex>
                    </SplitItem>
                    <SplitItem>
                        <Pagination
                            itemCount={filteredVirtualMachineComponentsTableData.length}
                            perPage={perPage}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </SplitItem>
                </Split>
                <VirtualMachineComponentsPageTable
                    tableState={tableState}
                    getSortParams={getSortParams}
                    onClearFilters={onClearFilters}
                />
            </div>
        </PageSection>
    );
}

export default VirtualMachinePageComponents;
