import React from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { TableUIState } from 'utils/getTableUIState';

import type { PackageTableRow } from '../aggregateUtils';
import { COMPONENT_SORT_FIELD } from '../../utils/sortFields';

export type VirtualMachinePackagesTableProps = {
    tableState: TableUIState<PackageTableRow>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function VirtualMachinePackagesTable({
    tableState,
    getSortParams,
    onClearFilters,
}: VirtualMachinePackagesTableProps) {
    const colSpan = 3;

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={false}
        >
            <Thead>
                <Tr>
                    <Th sort={getSortParams(COMPONENT_SORT_FIELD)}>Name</Th>
                    <Th>Status</Th>
                    <Th>Version</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                errorProps={{
                    title: 'There was an error loading results',
                }}
                emptyProps={{
                    message: 'No packages were detected for this virtual machine',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((packageRow) => {
                            return (
                                <Tr key={packageRow.name}>
                                    <Td dataLabel="Name">{packageRow.name} </Td>
                                    <Td dataLabel="Status">
                                        {packageRow.isScannable ? 'Scanned' : 'Not scanned'}
                                    </Td>
                                    <Td dataLabel="Version">{packageRow.version}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default VirtualMachinePackagesTable;
