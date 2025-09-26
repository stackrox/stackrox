import React, { useCallback, useMemo } from 'react';
import {
    Flex,
    PageSection,
    Pagination,
    pluralize,
    Skeleton,
    Split,
    SplitItem,
    Title,
} from '@patternfly/react-core';

import { DynamicTableLabel } from 'Components/DynamicIcon';
import { DEFAULT_VM_PAGE_SIZE } from 'Containers/Vulnerabilities/constants';
import {
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import { getVirtualMachine } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import { getHasSearchApplied } from 'utils/searchUtils';
import {
    getVirtualMachineCveTableData,
    applyVirtualMachineCveTableFilters,
} from '../aggregateUtils';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import VirtualMachineVulnerabilitiesTable from './VirtualMachineVulnerabilitiesTable';

export type VirtualMachinePageVulnerabilitiesProps = {
    virtualMachineId: string;
};

const searchFilterConfig = [
    virtualMachineCVESearchFilterConfig,
    virtualMachineComponentSearchFilterConfig,
];

function VirtualMachinePageVulnerabilities({
    virtualMachineId,
}: VirtualMachinePageVulnerabilitiesProps) {
    const fetchVirtualMachines = useCallback(
        () => getVirtualMachine(virtualMachineId),
        [virtualMachineId]
    );

    const { data, isLoading, error } = useRestQuery(fetchVirtualMachines);
    const pagination = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { page, perPage, setPage, setPerPage } = pagination;
    const { searchFilter, setSearchFilter } = useURLSearch();
    const isFiltered = getHasSearchApplied(searchFilter);

    const virtualMachineTableData = useMemo(() => getVirtualMachineCveTableData(data), [data]);

    const filteredVirtualMachineTableData = useMemo(
        () => applyVirtualMachineCveTableFilters(virtualMachineTableData, searchFilter),
        [virtualMachineTableData, searchFilter]
    );

    const paginatedVirtualMachineTableData = useMemo(() => {
        const totalRows = filteredVirtualMachineTableData.length;
        const maxPage = Math.max(1, Math.ceil(totalRows / perPage) || 1);
        const safePage = Math.min(page, maxPage);

        const start = (safePage - 1) * perPage;
        const end = start + perPage;
        return filteredVirtualMachineTableData.slice(start, end);
    }, [filteredVirtualMachineTableData, page, perPage]);

    const tableState = getTableUIState({
        isLoading,
        data: paginatedVirtualMachineTableData,
        error,
        searchFilter,
    });

    function onClearFilters() {
        setSearchFilter({});
        setPage(1);
    }

    return (
        <PageSection variant="light" isFilled padding={{ default: 'padding' }}>
            <AdvancedFiltersToolbar
                className="pf-v5-u-px-sm pf-v5-u-pb-0"
                searchFilter={searchFilter}
                searchFilterConfig={searchFilterConfig}
                onFilterChange={(newFilter) => {
                    setSearchFilter(newFilter);
                    setPage(1, 'replace');
                }}
            />
            <div className="pf-v5-u-flex-grow-1 pf-v5-u-background-color-100 pf-v5-u-p-lg">
                <Split className="pf-v5-u-pb-lg pf-v5-u-align-items-baseline">
                    <SplitItem isFilled>
                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                            <Title headingLevel="h2">
                                {!isLoading ? (
                                    `${pluralize(filteredVirtualMachineTableData.length, 'result')} found`
                                ) : (
                                    <Skeleton screenreaderText="Loading node vulnerability count" />
                                )}
                            </Title>
                            {isFiltered && <DynamicTableLabel />}
                        </Flex>
                    </SplitItem>
                    <SplitItem>
                        <Pagination
                            itemCount={filteredVirtualMachineTableData.length}
                            perPage={perPage}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </SplitItem>
                </Split>
                <VirtualMachineVulnerabilitiesTable
                    tableState={tableState}
                    onClearFilters={onClearFilters}
                />
            </div>
        </PageSection>
    );
}

export default VirtualMachinePageVulnerabilities;
