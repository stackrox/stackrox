import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLSortResult } from 'hooks/useURLSort';
import type { TableUIState } from 'utils/getTableUIState';

import type { ComponentTableRow } from '../aggregateUtils';
import { COMPONENT_SORT_FIELD } from '../../utils/sortFields';

export type VirtualMachineComponentsPageTableProps = {
    tableState: TableUIState<ComponentTableRow>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function VirtualMachineComponentsPageTable({
    tableState,
    getSortParams,
    onClearFilters,
}: VirtualMachineComponentsPageTableProps) {
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
                    <Th>Version</Th>
                    <Th>Status</Th>
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
                        {data.map((componentRow) => {
                            return (
                                <Tr key={`${componentRow.name}-${componentRow.version}`}>
                                    <Td dataLabel="Name">{componentRow.name} </Td>
                                    <Td dataLabel="Version">{componentRow.version}</Td>
                                    <Td dataLabel="Status">
                                        {componentRow.isScannable ? 'Scanned' : 'Not scanned'}
                                    </Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default VirtualMachineComponentsPageTable;
