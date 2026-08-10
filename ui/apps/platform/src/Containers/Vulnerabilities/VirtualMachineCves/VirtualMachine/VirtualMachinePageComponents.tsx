import { useCallback } from 'react';
import {
    PageSection,
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import type {
    OnSearchCallback,
    SelectSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';
import SearchFilterSelectInclusive from 'Components/CompoundSearchFilter/components/SearchFilterSelectInclusive';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { listVMComponents } from 'services/VirtualMachineService';
import type { VMComponentScanStatus } from 'services/VirtualMachineService';
import { DEFAULT_VM_PAGE_SIZE, sourceTypeLabels } from '../../constants';
import { getTableUIState } from 'utils/getTableUIState';
import { virtualMachineComponentSearchFilterConfig } from '../../searchFilterConfig';
import { scanStatuses } from '../../types';
import { COMPONENT_SORT_FIELD } from '../../utils/sortFields';

const sortFields = [COMPONENT_SORT_FIELD];

const defaultSortOption = { field: COMPONENT_SORT_FIELD, direction: 'asc' } as const;

const searchFilterConfig = [virtualMachineComponentSearchFilterConfig];

const attributeForScanStatus: SelectSearchFilterAttribute = {
    displayName: 'Scan status',
    filterChipLabel: 'Scan status',
    searchTerm: 'Scan Status',
    inputType: 'select',
    inputProps: {
        options: scanStatuses.map((label) => ({ label, value: label })),
    },
};

const scanStatusDisplayMap: Record<VMComponentScanStatus, string> = {
    SCANNED: 'Scanned',
    NOT_SCANNED: 'Not scanned',
    SCAN_PENDING: 'Scan pending',
    CPE_MISSING: 'CPE missing',
    REPO_UNKNOWN: 'Repo unknown',
};

export type VirtualMachinePageComponentsProps = {
    virtualMachineId: string;
};

function VirtualMachinePageComponents({ virtualMachineId }: VirtualMachinePageComponentsProps) {
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const fetchComponents = useCallback(
        () => listVMComponents(virtualMachineId, { searchFilter, page, perPage, sortOption }),
        [virtualMachineId, searchFilter, page, perPage, sortOption]
    );
    const { data, isLoading, error } = useRestQuery(fetchComponents);

    const tableState = getTableUIState({
        isLoading,
        data: data?.components ?? [],
        error,
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

    const colSpan = 4;

    return (
        <PageSection isFilled>
            <Toolbar>
                <ToolbarContent>
                    <CompoundSearchFilter
                        config={searchFilterConfig}
                        searchFilter={searchFilter}
                        onSearch={onSearch}
                    />
                    <SearchFilterSelectInclusive
                        attribute={attributeForScanStatus}
                        isSeparate
                        onSearch={onSearch}
                        searchFilter={searchFilter}
                    />
                    <ToolbarGroup className="pf-v6-u-w-100">
                        <CompoundSearchFilterLabels
                            attributesSeparateFromConfig={[attributeForScanStatus]}
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onFilterChange={setSearchFilter}
                        />
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>
            <Pagination
                itemCount={data?.totalCount ?? 0}
                perPage={perPage}
                page={page}
                onSetPage={(_, newPage) => setPage(newPage)}
                onPerPageSelect={(_, newPerPage) => {
                    setPerPage(newPerPage);
                }}
            />
            <Table
                borders={tableState.type === 'COMPLETE'}
                variant="compact"
                aria-live="polite"
                aria-busy={isLoading}
            >
                <Thead>
                    <Tr>
                        <Th sort={getSortParams(COMPONENT_SORT_FIELD)}>Name</Th>
                        <Th>Version</Th>
                        <Th>Status</Th>
                        <Th>Source</Th>
                    </Tr>
                </Thead>
                <TbodyUnified
                    tableState={tableState}
                    colSpan={colSpan}
                    errorProps={{
                        title: 'There was an error loading results',
                    }}
                    emptyProps={{
                        message: 'No components were detected for this virtual machine',
                    }}
                    filteredEmptyProps={{ onClearFilters }}
                    renderer={({ data }) => (
                        <Tbody>
                            {data.map((componentRow) => (
                                <Tr key={componentRow.id}>
                                    <Td dataLabel="Name">{componentRow.name}</Td>
                                    <Td dataLabel="Version">{componentRow.version}</Td>
                                    <Td dataLabel="Status">
                                        {scanStatusDisplayMap[componentRow.scanStatus] ??
                                            componentRow.scanStatus}
                                    </Td>
                                    <Td dataLabel="Source">
                                        {sourceTypeLabels[componentRow.source]}
                                    </Td>
                                </Tr>
                            ))}
                        </Tbody>
                    )}
                />
            </Table>
        </PageSection>
    );
}

export default VirtualMachinePageComponents;
