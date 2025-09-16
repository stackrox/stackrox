import React from 'react';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { TableUIState } from 'utils/getTableUIState';

import type { CVEWithAffectedComponents } from '../aggregateUtils';

export type VirtualMachineVulnerabilitiesTableProps = {
    tableState: TableUIState<CVEWithAffectedComponents>;
};

function VirtualMachineVulnerabilitiesTable({
    tableState,
}: VirtualMachineVulnerabilitiesTableProps) {
    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={false}
        >
            <Thead>
                <Tr>
                    <Th>CVE</Th>
                    <Th>Severity</Th>
                    <Th>CVE status</Th>
                    <Th>CVSS</Th>
                    <Th>EPSS probability</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={7}
                errorProps={{
                    title: 'There was an error loading results',
                }}
                emptyProps={{
                    message: 'No CVEs were detected for this virtual machine',
                }}
                renderer={({ data }) => (
                    <Tbody>
                        {data.map((virtualMachine) => {
                            return (
                                <Tr key={virtualMachine.cve}>
                                    <Td dataLabel="CVE">{virtualMachine.cve} </Td>
                                    <Td dataLabel="Severity">
                                        <VulnerabilitySeverityIconText
                                            severity={virtualMachine.severity}
                                        />
                                    </Td>
                                    <Td dataLabel="CVE status">ROX-30535</Td>
                                    <Td dataLabel="CVSS">ROX-30535</Td>
                                    <Td dataLabel="EPSS probability">ROX-30535</Td>
                                    <Td dataLabel="Affected components">ROX-30535</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                )}
            />
        </Table>
    );
}

export default VirtualMachineVulnerabilitiesTable;
