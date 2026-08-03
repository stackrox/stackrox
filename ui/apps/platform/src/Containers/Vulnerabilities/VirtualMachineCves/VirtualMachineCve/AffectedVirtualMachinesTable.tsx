import { Truncate } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import pluralize from 'pluralize';
import { Link } from 'react-router-dom-v5-compat';

import CvssFormatted from 'Components/CvssFormatted';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useSet from 'hooks/useSet';
import type { TableUIState } from 'utils/getTableUIState';

import type { VMCVEAffectedVMRow } from 'services/VirtualMachineService';

import { getVirtualMachineEntityPagePath } from '../../utils/searchUtils';

export type AffectedVirtualMachinesTableProps = {
    tableState: TableUIState<VMCVEAffectedVMRow>;
    onClearFilters: () => void;
};

function AffectedVirtualMachinesTable({
    tableState,
    onClearFilters,
}: AffectedVirtualMachinesTableProps) {
    const colSpan = 7;
    const expandedRowSet = useSet<string>();

    return (
        <Table borders={tableState.type === 'COMPLETE'} variant="compact" aria-live="polite">
            <Thead noWrap>
                <Tr>
                    <Th screenReaderText="Row expansion" />
                    <Th>Virtual machine</Th>
                    <Th>CVE severity</Th>
                    <Th>CVE status</Th>
                    <Th>CVSS</Th>
                    <Th>Guest OS</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{
                    message: 'There are no virtual machines affected by this CVE',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((virtualMachine, rowIndex) => {
                        const isExpanded = expandedRowSet.has(virtualMachine.vmId);

                        return (
                            <Tbody key={virtualMachine.vmId} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () =>
                                                expandedRowSet.toggle(virtualMachine.vmId),
                                        }}
                                    />
                                    <Td dataLabel="Virtual machine">
                                        <Link
                                            to={getVirtualMachineEntityPagePath(
                                                'VirtualMachine',
                                                virtualMachine.vmId
                                            )}
                                        >
                                            <Truncate
                                                position="middle"
                                                content={virtualMachine.vmName}
                                            />
                                        </Link>
                                    </Td>
                                    <Td dataLabel="CVE severity" modifier="nowrap">
                                        <VulnerabilitySeverityIconText
                                            severity={virtualMachine.severity}
                                        />
                                    </Td>
                                    <Td dataLabel="CVE status" modifier="nowrap">
                                        <VulnerabilityFixableIconText
                                            isFixable={virtualMachine.isFixable}
                                        />
                                    </Td>
                                    <Td dataLabel="CVSS" modifier="nowrap">
                                        <CvssFormatted cvss={virtualMachine.cvss} />
                                    </Td>
                                    <Td dataLabel="Guest OS">
                                        <Truncate
                                            position="middle"
                                            content={virtualMachine.guestOs}
                                        />
                                    </Td>
                                    <Td dataLabel="Affected components">
                                        {`${virtualMachine.affectedComponentCount} ${pluralize('component', virtualMachine.affectedComponentCount)}`}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={colSpan - 1}>
                                        <ExpandableRowContent>
                                            Affected component details coming soon
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default AffectedVirtualMachinesTable;
