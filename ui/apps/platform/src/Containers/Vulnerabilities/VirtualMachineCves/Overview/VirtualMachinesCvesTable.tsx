import React, { useCallback } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useRestQuery from 'hooks/useRestQuery';
import { listVirtualMachines } from 'services/VirtualMachineService';
import { getTableUIState } from 'utils/getTableUIState';

function VirtualMachinesCvesTable() {
    const fetchVirtualMachines = useCallback(() => listVirtualMachines(), []);

    const { data, isLoading, error } = useRestQuery(fetchVirtualMachines);

    const tableState = getTableUIState({
        isLoading,
        data: data ?? [],
        error,
        searchFilter: {},
    });

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={false}
        >
            <Thead>
                <Tr>
                    <Th>Virtual machine</Th>
                    <Th>CVEs by severity</Th>
                    <Th>Guest OS</Th>
                    <Th>Cluster</Th>
                    <Th>Namespace</Th>
                    <Th>Pod</Th>
                    <Th>Created</Th>
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
                            return (
                                <Tr key={virtualMachine.id}>
                                    <Td dataLabel="Virtual machine">
                                        <Link to={''}>{virtualMachine.name}</Link>
                                    </Td>
                                    <Td dataLabel="CVEs by severity">placeholder for ROX-30528</Td>
                                    <Td dataLabel="Guest OS">ROX-30535</Td>
                                    <Td dataLabel="Cluster">{virtualMachine.clusterName}</Td>
                                    <Td dataLabel="Namespace">{virtualMachine.namespace}</Td>
                                    <Td dataLabel="Pod">ROX-30535</Td>
                                    <Td dataLabel="Created">ROX-30535</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default VirtualMachinesCvesTable;
