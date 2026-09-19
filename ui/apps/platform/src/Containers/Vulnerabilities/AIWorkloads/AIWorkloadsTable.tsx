import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Label } from '@patternfly/react-core';

import type { AIWorkload } from 'services/AIWorkloadService';

type AIWorkloadsTableProps = {
    workloads: AIWorkload[];
};

function AIWorkloadsTable({ workloads }: AIWorkloadsTableProps) {
    return (
        <Table variant="compact">
            <Thead>
                <Tr>
                    <Th>Name</Th>
                    <Th>Type</Th>
                    <Th>Model Format</Th>
                    <Th>Namespace</Th>
                    <Th>Cluster</Th>
                    <Th>GPU</Th>
                    <Th>Auth</Th>
                </Tr>
            </Thead>
            <Tbody>
                {workloads.map((workload) => (
                    <Tr key={workload.id}>
                        <Td dataLabel="Name">{workload.name}</Td>
                        <Td dataLabel="Type">
                            <Label color={workload.workloadType === 'INFERENCE' ? 'blue' : 'green'}>
                                {workload.workloadType}
                            </Label>
                        </Td>
                        <Td dataLabel="Model Format">{workload.modelFormat || '-'}</Td>
                        <Td dataLabel="Namespace">{workload.namespace}</Td>
                        <Td dataLabel="Cluster">{workload.clusterName}</Td>
                        <Td dataLabel="GPU">{workload.gpuRequests || '-'}</Td>
                        <Td dataLabel="Auth">
                            <Label color={workload.authEnabled ? 'green' : 'red'}>
                                {workload.authEnabled ? 'Enabled' : 'Disabled'}
                            </Label>
                        </Td>
                    </Tr>
                ))}
            </Tbody>
        </Table>
    );
}

export default AIWorkloadsTable;
