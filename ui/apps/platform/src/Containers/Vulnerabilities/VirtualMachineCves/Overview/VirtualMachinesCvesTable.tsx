import { useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Flex, FlexItem, Pagination } from '@patternfly/react-core';
import { InnerScrollContainer, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import ColumnManagementButton from 'Components/ColumnManagementButton';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    useManagedColumns,
} from 'hooks/useManagedColumns';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { listVirtualMachines } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

import {
    getVirtualMachineScannedComponentsCount,
    getVirtualMachineSeveritiesCount,
} from '../aggregateUtils';
import AdvancedFiltersToolbar from '../../components/AdvancedFiltersToolbar';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import {
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
} from '../../searchFilterConfig';
import { getVirtualMachineEntityPagePath } from '../../utils/searchUtils';
import { VIRTUAL_MACHINE_SORT_FIELD } from '../../utils/sortFields';

const searchFilterConfig = [
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
];

export const sortFields = [VIRTUAL_MACHINE_SORT_FIELD];

export const defaultSortOption = { field: VIRTUAL_MACHINE_SORT_FIELD, direction: 'asc' } as const;

export const defaultColumns = {
    virtualMachine: {
        title: 'Virtual machine',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cvesBySeverity: {
        title: 'CVEs by severity',
        isShownByDefault: true,
    },
    cluster: {
        title: 'Cluster',
        isShownByDefault: true,
    },
    namespace: {
        title: 'Namespace',
        isShownByDefault: true,
    },
    scannedComponents: {
        title: 'Scanned components',
        isShownByDefault: true,
    },
    scanTime: {
        title: 'Scan time',
        isShownByDefault: true,
    },
} as const;

function VirtualMachinesCvesTable() {
    const managedColumnState = useManagedColumns('VirtualMachinesCvesTable', defaultColumns);
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const getVisibilityClass = generateVisibilityForColumns(managedColumnState.columns);
    const hiddenColumnCount = getHiddenColumnCount(managedColumnState.columns);
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

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
            <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                <FlexItem fullWidth={{ default: 'fullWidth' }}>
                    <AdvancedFiltersToolbar
                        defaultSearchFilterEntity="Virtual Machine"
                        includeCveSeverityFilters={false}
                        includeCveStatusFilters={false}
                        searchFilter={searchFilter}
                        searchFilterConfig={searchFilterConfig}
                        onFilterChange={(newFilter) => {
                            setSearchFilter(newFilter);
                            setPage(1, 'replace');
                        }}
                    />
                </FlexItem>
                <ColumnManagementButton
                    columnConfig={managedColumnState.columns}
                    onApplyColumns={managedColumnState.setVisibility}
                />
                <Pagination
                    itemCount={data?.totalCount ?? 0}
                    perPage={perPage}
                    page={page}
                    onSetPage={(_, newPage) => setPage(newPage)}
                    onPerPageSelect={(_, newPerPage) => {
                        setPerPage(newPerPage);
                    }}
                />
            </Flex>
            <InnerScrollContainer>
                <Table
                    borders={tableState.type === 'COMPLETE'}
                    variant="compact"
                    aria-live="polite"
                    aria-busy={false}
                >
                    <Thead>
                        <Tr>
                            <Th
                                className={getVisibilityClass('virtualMachine')}
                                sort={getSortParams('Virtual Machine Name')}
                                modifier="fitContent"
                            >
                                Virtual machine
                            </Th>
                            <Th className={getVisibilityClass('cvesBySeverity')}>
                                CVEs by severity
                            </Th>
                            <Th className={getVisibilityClass('cluster')}>Cluster</Th>
                            <Th className={getVisibilityClass('namespace')}>Namespace</Th>
                            <Th className={getVisibilityClass('scannedComponents')}>
                                Scanned components
                            </Th>
                            <Th className={getVisibilityClass('scanTime')}>Scan time</Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={colSpan}
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

                                    const scanTime = virtualMachine?.scan?.scanTime;
                                    return (
                                        <Tr key={virtualMachine.id}>
                                            <Td
                                                className={getVisibilityClass('virtualMachine')}
                                                dataLabel="Virtual machine"
                                                modifier="nowrap"
                                            >
                                                <Link
                                                    to={getVirtualMachineEntityPagePath(
                                                        'VirtualMachine',
                                                        virtualMachine.id
                                                    )}
                                                >
                                                    {virtualMachine.name}
                                                </Link>
                                            </Td>
                                            <Td
                                                className={getVisibilityClass('cvesBySeverity')}
                                                dataLabel="CVEs by severity"
                                            >
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
                                                    entity="virtual machine"
                                                />
                                            </Td>
                                            <Td
                                                className={getVisibilityClass('cluster')}
                                                dataLabel="Cluster"
                                            >
                                                {virtualMachine.clusterName}
                                            </Td>
                                            <Td
                                                className={getVisibilityClass('namespace')}
                                                dataLabel="Namespace"
                                            >
                                                {virtualMachine.namespace}
                                            </Td>
                                            <Td
                                                className={getVisibilityClass('scannedComponents')}
                                                dataLabel="Scanned components"
                                            >
                                                {getVirtualMachineScannedComponentsCount(
                                                    virtualMachine
                                                )}
                                            </Td>
                                            <Td
                                                className={getVisibilityClass('scanTime')}
                                                dataLabel="Scan time"
                                            >
                                                {typeof scanTime === 'string' ? (
                                                    <DateDistance date={scanTime} />
                                                ) : (
                                                    'Not available'
                                                )}
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        )}
                    />
                </Table>
            </InnerScrollContainer>
        </>
    );
}

export default VirtualMachinesCvesTable;
