import { useMemo } from 'react';
import { Flex, PageSection, Pagination } from '@patternfly/react-core';

import ColumnManagementButton from 'Components/ColumnManagementButton';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import type { UseUrlSearchReturn } from 'hooks/useURLSearch';
import type { UseURLSortResult } from 'hooks/useURLSort';
import { useManagedColumns } from 'hooks/useManagedColumns';
import type { VirtualMachine } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import {
    applyVirtualMachineCveTableFilters,
    applyVirtualMachineCveTableSort,
    getVirtualMachineCveSeverityStatusCounts,
    getVirtualMachineCveTableData,
} from '../aggregateUtils';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import BySeveritySummaryCard from '../../components/BySeveritySummaryCard';
import CvesByStatusSummaryCard from '../../components/CvesByStatusSummaryCard';
import { SummaryCard, SummaryCardLayout } from '../../components/SummaryCardLayout';
import VirtualMachineScanScopeAlert from '../components/VirtualMachineScanScopeAlert';
import {
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
} from '../../searchFilterConfig';
import {
    getHiddenSeverities,
    getHiddenStatuses,
    parseQuerySearchFilter,
} from '../../utils/searchUtils';
import VirtualMachineVulnerabilitiesTable, {
    defaultColumns,
    tableId,
} from './VirtualMachineVulnerabilitiesTable';

// Currently we need all vm info to be fetched in the root component, hence this being passed in
// there will likely be a call specific to this table in the future that should be made here
export type VirtualMachinePageVulnerabilitiesProps = {
    virtualMachine: VirtualMachine | undefined;
    isLoadingVirtualMachine: boolean;
    errorVirtualMachine: Error | undefined;
    urlSearch: UseUrlSearchReturn;
    urlSorting: UseURLSortResult;
    urlPagination: UseURLPaginationResult;
};

const searchFilterConfig = [
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
];

function VirtualMachinePageVulnerabilities({
    virtualMachine,
    isLoadingVirtualMachine,
    errorVirtualMachine,
    urlSearch,
    urlSorting,
    urlPagination,
}: VirtualMachinePageVulnerabilitiesProps) {
    const { searchFilter, setSearchFilter } = urlSearch;
    const { sortOption, getSortParams } = urlSorting;
    const { page, perPage, setPage, setPerPage } = urlPagination;
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const hiddenStatuses = getHiddenStatuses(querySearchFilter);
    const hiddenSeverities = getHiddenSeverities(querySearchFilter);

    const managedColumnState = useManagedColumns(tableId, defaultColumns);

    const virtualMachineTableData = useMemo(
        () => getVirtualMachineCveTableData(virtualMachine),
        [virtualMachine]
    );

    const filteredVirtualMachineTableData = useMemo(
        () => applyVirtualMachineCveTableFilters(virtualMachineTableData, searchFilter),
        [virtualMachineTableData, searchFilter]
    );

    const sortedVirtualMachineTableData = useMemo(
        () =>
            applyVirtualMachineCveTableSort(
                filteredVirtualMachineTableData,
                Array.isArray(sortOption) ? sortOption[0].field : sortOption.field,
                Array.isArray(sortOption) ? sortOption[0].reversed : sortOption.reversed
            ),
        [filteredVirtualMachineTableData, sortOption]
    );

    const paginatedVirtualMachineTableData = useMemo(() => {
        const totalRows = sortedVirtualMachineTableData.length;
        const maxPage = Math.max(1, Math.ceil(totalRows / perPage) || 1);
        const safePage = Math.min(page, maxPage);

        const start = (safePage - 1) * perPage;
        const end = start + perPage;
        return sortedVirtualMachineTableData.slice(start, end);
    }, [sortedVirtualMachineTableData, page, perPage]);

    const tableState = getTableUIState({
        isLoading: isLoadingVirtualMachine,
        data: paginatedVirtualMachineTableData,
        error: errorVirtualMachine,
        searchFilter,
    });

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    return (
        <PageSection variant="light" isFilled padding={{ default: 'padding' }}>
            <VirtualMachineScanScopeAlert />
            <AdvancedFiltersToolbar
                className="pf-v5-u-px-sm pf-v5-u-pb-0"
                defaultSearchFilterEntity="CVE"
                searchFilter={searchFilter}
                searchFilterConfig={searchFilterConfig}
                onFilterChange={(newFilter) => {
                    setSearchFilter(newFilter);
                    setPage(1, 'replace');
                }}
            />
            <SummaryCardLayout isLoading={isLoadingVirtualMachine} error={errorVirtualMachine}>
                <SummaryCard
                    loadingText={'Loading virtual machine CVEs by severity summary'}
                    data={filteredVirtualMachineTableData}
                    renderer={({ data }) => (
                        <BySeveritySummaryCard
                            title="CVEs by severity"
                            severityCounts={getVirtualMachineCveSeverityStatusCounts(data)}
                            hiddenSeverities={hiddenSeverities}
                        />
                    )}
                />
                <SummaryCard
                    loadingText={'Loading virtual machine CVEs by status summary'}
                    data={filteredVirtualMachineTableData}
                    renderer={({ data }) => (
                        <CvesByStatusSummaryCard
                            cveStatusCounts={getVirtualMachineCveSeverityStatusCounts(data)}
                            hiddenStatuses={hiddenStatuses}
                        />
                    )}
                />
            </SummaryCardLayout>
            <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                <ColumnManagementButton
                    columnConfig={managedColumnState.columns}
                    onApplyColumns={managedColumnState.setVisibility}
                />
                <Pagination
                    itemCount={filteredVirtualMachineTableData.length}
                    perPage={perPage}
                    page={page}
                    onSetPage={(_, newPage) => setPage(newPage)}
                    onPerPageSelect={(_, newPerPage) => {
                        setPerPage(newPerPage);
                    }}
                />
            </Flex>
            <VirtualMachineVulnerabilitiesTable
                onClearFilters={onClearFilters}
                getSortParams={getSortParams}
                tableState={tableState}
                tableConfig={managedColumnState.columns}
            />
        </PageSection>
    );
}

export default VirtualMachinePageVulnerabilities;
