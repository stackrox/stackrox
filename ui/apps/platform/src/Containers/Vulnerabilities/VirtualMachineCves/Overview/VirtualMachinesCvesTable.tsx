import React, { useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import {
    Flex,
    Pagination,
    pluralize,
    Skeleton,
    Split,
    SplitItem,
    Title,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import DateDistance from 'Components/DateDistance';
import { DynamicTableLabel } from 'Components/DynamicIcon';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import {
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { listVirtualMachines } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';
import { getHasSearchApplied } from 'utils/searchUtils';

import {
    getVirtualMachineScannedPackagesCount,
    getVirtualMachineSeveritiesCount,
} from '../aggregateUtils';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { getVirtualMachineEntityPagePath } from '../../utils/searchUtils';
import { VIRTUAL_MACHINE_SORT_FIELD } from '../../utils/sortFields';

const searchFilterConfig = [
    virtualMachinesSearchFilterConfig,
    virtualMachinesClusterSearchFilterConfig,
];

export const sortFields = [VIRTUAL_MACHINE_SORT_FIELD];

export const defaultSortOption = { field: VIRTUAL_MACHINE_SORT_FIELD, direction: 'asc' } as const;

function VirtualMachinesCvesTable() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const isFiltered = getHasSearchApplied(searchFilter);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const fetchVirtualMachines = useCallback(
        () => listVirtualMachines({ searchFilter, page, perPage, sortOption }),
        [searchFilter, page, perPage, sortOption]
    );
    const { data, isLoading, error } = useRestQuery(fetchVirtualMachines);
    const tableState = getTableUIState({
        isLoading,
        data: data?.virtualMachines ?? [],
        error,
        searchFilter,
    });

    return (
        <>
            <AdvancedFiltersToolbar
                className="pf-v5-u-px-sm pf-v5-u-pb-0"
                includeCveSeverityFilters={false}
                includeCveStatusFilters={false}
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
                                    `${pluralize(data?.totalCount ?? 0, 'result')} found`
                                ) : (
                                    <Skeleton screenreaderText="Loading virtual machine count" />
                                )}
                            </Title>
                            {isFiltered && <DynamicTableLabel />}
                        </Flex>
                    </SplitItem>
                    <SplitItem>
                        <Pagination
                            itemCount={data?.totalCount ?? 0}
                            perPage={perPage}
                            page={page}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPerPage(newPerPage);
                            }}
                        />
                    </SplitItem>
                </Split>
                <Table
                    borders={tableState.type === 'COMPLETE'}
                    variant="compact"
                    aria-live="polite"
                    aria-busy={false}
                >
                    <Thead>
                        <Tr>
                            <Th sort={getSortParams('Virtual Machine Name')}>Virtual machine</Th>
                            <Th>CVEs by severity</Th>
                            <Th>Cluster</Th>
                            <Th>Namespace</Th>
                            <Th modifier="fitContent">Scanned packages</Th>
                            <Th>Last updated</Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={7}
                        errorProps={{
                            title: 'There was an error loading results',
                        }}
                        emptyProps={{
                            message: 'No CVEs have been detected',
                        }}
                        renderer={({ data }) => (
                            <Tbody>
                                {data.map((virtualMachine) => {
                                    const virtualMachineSeverityCounts =
                                        getVirtualMachineSeveritiesCount(virtualMachine);
                                    return (
                                        <Tr key={virtualMachine.id}>
                                            <Td dataLabel="Virtual machine" modifier="nowrap">
                                                <Link
                                                    to={getVirtualMachineEntityPagePath(
                                                        'VirtualMachine',
                                                        virtualMachine.id
                                                    )}
                                                >
                                                    {virtualMachine.name}
                                                </Link>
                                            </Td>
                                            <Td dataLabel="CVEs by severity">
                                                <SeverityCountLabels
                                                    criticalCount={
                                                        virtualMachineSeverityCounts.CRITICAL_VULNERABILITY_SEVERITY
                                                    }
                                                    importantCount={
                                                        virtualMachineSeverityCounts.IMPORTANT_VULNERABILITY_SEVERITY
                                                    }
                                                    moderateCount={
                                                        virtualMachineSeverityCounts.MODERATE_VULNERABILITY_SEVERITY
                                                    }
                                                    lowCount={
                                                        virtualMachineSeverityCounts.LOW_VULNERABILITY_SEVERITY
                                                    }
                                                    unknownCount={
                                                        virtualMachineSeverityCounts.UNKNOWN_VULNERABILITY_SEVERITY
                                                    }
                                                />
                                            </Td>
                                            <Td dataLabel="Cluster">
                                                {virtualMachine.clusterName}
                                            </Td>
                                            <Td dataLabel="Namespace">
                                                {virtualMachine.namespace}
                                            </Td>
                                            <Td dataLabel="Scanned packages">
                                                {getVirtualMachineScannedPackagesCount(
                                                    virtualMachine
                                                )}
                                            </Td>
                                            <Td dataLabel="Last updated">
                                                <DateDistance date={virtualMachine.lastUpdated} />
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        )}
                    />
                </Table>
            </div>
        </>
    );
}

export default VirtualMachinesCvesTable;
