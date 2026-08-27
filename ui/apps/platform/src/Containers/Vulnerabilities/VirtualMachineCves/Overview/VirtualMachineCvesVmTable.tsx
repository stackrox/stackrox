import { useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { ToolbarItem } from '@patternfly/react-core';
import { InnerScrollContainer, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import ColumnManagementButton from 'Components/ColumnManagementButton';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { generateVisibilityForColumns, getHiddenColumnCount } from 'hooks/useManagedColumns';
import type { ManagedColumns } from 'hooks/useManagedColumns';
import useFeatureFlags from 'hooks/useFeatureFlags';
import useRestQuery from 'hooks/useRestQuery';
import type { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { listVMs } from 'services/VirtualMachineService';
import type { SearchFilter } from 'types/search';
import { getTableUIState } from 'utils/getTableUIState';

import SeverityCountLabels from '../../components/SeverityCountLabels';
import TableEntityToolbar from '../../components/TableEntityToolbar';
import { getVirtualMachineEntityPagePath } from '../../utils/searchUtils';
import {
    CLUSTER_SORT_FIELD,
    NAMESPACE_SORT_FIELD,
    VIRTUAL_MACHINE_SORT_FIELD,
} from '../../utils/sortFields';
import VirtualMachineCvesFilterToolbar from './VirtualMachineCvesFilterToolbar';
import VirtualMachineCvesToggleGroup from './VirtualMachineCvesToggleGroup';
import VirtualMachinesCvesTableLegacy from './VirtualMachinesCvesTableLegacy';

const sortFields = [VIRTUAL_MACHINE_SORT_FIELD, CLUSTER_SORT_FIELD, NAMESPACE_SORT_FIELD];

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

type VirtualMachineCvesVmTableProps = {
    searchFilter: SearchFilter;
    pagination: UseURLPaginationResult;
    managedColumnState: ManagedColumns<keyof typeof defaultColumns>;
    isFiltered: boolean;
    onClearFilters: () => void;
};

type VirtualMachineCvesVmTableEnhancedProps = VirtualMachineCvesVmTableProps;

function VirtualMachineCvesVmTableEnhanced({
    searchFilter,
    pagination,
    managedColumnState,
    isFiltered,
    onClearFilters,
}: VirtualMachineCvesVmTableEnhancedProps) {
    const { page, perPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => pagination.setPage(1, 'replace'),
    });

    const getVisibilityClass = generateVisibilityForColumns(managedColumnState.columns);
    const hiddenColumnCount = getHiddenColumnCount(managedColumnState.columns);
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

    const fetchVirtualMachines = useCallback(
        () => listVMs({ searchFilter, page, perPage, sortOption }),
        [searchFilter, page, perPage, sortOption]
    );
    const { data, isLoading, error } = useRestQuery(fetchVirtualMachines);

    const totalCount = data?.totalCount ?? 0;

    const tableState = getTableUIState({
        isLoading,
        data: data?.vms ?? [],
        error,
        searchFilter,
    });

    return (
        <>
            <TableEntityToolbar
                filterToolbar={<VirtualMachineCvesFilterToolbar />}
                entityToggleGroup={<VirtualMachineCvesToggleGroup />}
                pagination={pagination}
                tableRowCount={totalCount}
                isFiltered={isFiltered}
            >
                <ToolbarItem>
                    <ColumnManagementButton
                        columnConfig={managedColumnState.columns}
                        onApplyColumns={managedColumnState.setVisibility}
                    />
                </ToolbarItem>
            </TableEntityToolbar>
            <InnerScrollContainer>
                <Table
                    borders={tableState.type === 'COMPLETE'}
                    variant="compact"
                    aria-live="polite"
                    aria-busy={isLoading}
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
                            <Th
                                className={getVisibilityClass('cluster')}
                                sort={getSortParams(CLUSTER_SORT_FIELD)}
                            >
                                Cluster
                            </Th>
                            <Th
                                className={getVisibilityClass('namespace')}
                                sort={getSortParams(NAMESPACE_SORT_FIELD)}
                            >
                                Namespace
                            </Th>
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
                        filteredEmptyProps={{ onClearFilters }}
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

function VirtualMachineCvesVmTable(props: VirtualMachineCvesVmTableProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isEnhancedDataModelEnabled = isFeatureFlagEnabled(
        'ROX_VIRTUAL_MACHINES_ENHANCED_DATA_MODEL'
    );

    if (isEnhancedDataModelEnabled) {
        return <VirtualMachineCvesVmTableEnhanced {...props} />;
    }

    return <VirtualMachinesCvesTableLegacy />;
}

export default VirtualMachineCvesVmTable;
