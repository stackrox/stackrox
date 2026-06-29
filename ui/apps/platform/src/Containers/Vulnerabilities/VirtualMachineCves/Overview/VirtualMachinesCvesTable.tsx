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
import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import useURLSearch from 'hooks/useURLSearch';
import useURLSort from 'hooks/useURLSort';
import { listVMs } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

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
import VirtualMachinesCvesTableLegacy from './VirtualMachinesCvesTableLegacy';

const searchFilterConfig = [
    virtualMachinesClusterSearchFilterConfig,
    virtualMachinesNamespaceSearchFilterConfig,
    virtualMachinesSearchFilterConfig,
];

const sortFields = [VIRTUAL_MACHINE_SORT_FIELD];

const defaultSortOption = { field: VIRTUAL_MACHINE_SORT_FIELD, direction: 'asc' } as const;

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

function VirtualMachinesCvesTableEnhanced() {
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
        () => listVMs({ searchFilter, page, perPage, sortOption }),
        [searchFilter, page, perPage, sortOption]
    );
    const { data, isLoading, error } = useRestQuery(fetchVirtualMachines);
    const tableState = getTableUIState({
        isLoading,
        data: data?.vms ?? [],
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
                                    const counts = virtualMachine.cveSeverityCounts;
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
                                                    criticalCount={counts?.critical?.total ?? 0}
                                                    importantCount={counts?.important?.total ?? 0}
                                                    moderateCount={counts?.moderate?.total ?? 0}
                                                    lowCount={counts?.low?.total ?? 0}
                                                    unknownCount={counts?.unknown?.total ?? 0}
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
                                                {virtualMachine.componentScanCount
                                                    ? `${virtualMachine.componentScanCount.scanned} / ${virtualMachine.componentScanCount.total} scanned components`
                                                    : 'Not available'}
                                            </Td>
                                            <Td
                                                className={getVisibilityClass('scanTime')}
                                                dataLabel="Scan time"
                                            >
                                                {virtualMachine.scanTime ? (
                                                    <DateDistance date={virtualMachine.scanTime} />
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

function VirtualMachinesCvesTable() {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );

    if (isEnhancedDataModelEnabled) {
        return <VirtualMachinesCvesTableEnhanced />;
    }

    return <VirtualMachinesCvesTableLegacy />;
}

export default VirtualMachinesCvesTable;
