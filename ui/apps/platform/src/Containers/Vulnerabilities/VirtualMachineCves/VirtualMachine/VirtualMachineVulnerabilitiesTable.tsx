import React from 'react';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import ExpandRowTh from 'Components/ExpandRowTh';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import type { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import type { TableUIState } from 'utils/getTableUIState';

import type { CveTableRow } from '../aggregateUtils';
import {
    CVE_EPSS_PROBABILITY_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import { formatEpssProbabilityAsPercent } from '../../WorkloadCves/Tables/table.utils';
import VirtualMachineComponentsTable from './VirtualMachineComponentsTable';

export type VirtualMachineVulnerabilitiesTableProps = {
    tableState: TableUIState<CveTableRow>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function VirtualMachineVulnerabilitiesTable({
    tableState,
    getSortParams,
    onClearFilters,
}: VirtualMachineVulnerabilitiesTableProps) {
    const colSpan = 7;
    const expandedRowSet = useSet<string>();

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={false}
        >
            <Thead>
                <Tr>
                    <ExpandRowTh />
                    <Th sort={getSortParams(CVE_SORT_FIELD)}>CVE</Th>
                    <Th sort={getSortParams(CVE_SEVERITY_SORT_FIELD)}>Severity</Th>
                    <Th>CVE status</Th>
                    <Th sort={getSortParams(CVSS_SORT_FIELD)}>CVSS</Th>
                    <Th sort={getSortParams(CVE_EPSS_PROBABILITY_SORT_FIELD)}>EPSS probability</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                errorProps={{
                    title: 'There was an error loading results',
                }}
                emptyProps={{
                    message: 'No CVEs were detected for this virtual machine',
                }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((vulnerability, rowIndex) => {
                        const isExpanded = expandedRowSet.has(vulnerability.cve);
                        return (
                            <Tbody key={vulnerability.cve} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () =>
                                                expandedRowSet.toggle(vulnerability.cve),
                                        }}
                                    />
                                    <Td dataLabel="CVE">{vulnerability.cve} </Td>
                                    <Td dataLabel="Severity">
                                        <VulnerabilitySeverityIconText
                                            severity={vulnerability.severity}
                                        />
                                    </Td>
                                    <Td dataLabel="CVE status">
                                        <VulnerabilityFixableIconText
                                            isFixable={vulnerability.isFixable}
                                        />
                                    </Td>
                                    <Td dataLabel="CVSS">
                                        <CvssFormatted
                                            cvss={vulnerability.cvss}
                                            scoreVersion="v3"
                                        />
                                    </Td>
                                    <Td dataLabel="EPSS probability">
                                        {formatEpssProbabilityAsPercent(
                                            vulnerability.epssProbability
                                        )}
                                    </Td>
                                    <Td dataLabel="Affected components">
                                        {vulnerability.affectedComponents.length === 1
                                            ? vulnerability.affectedComponents[0].name
                                            : `${vulnerability.affectedComponents.length} components`}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={colSpan - 1}>
                                        <ExpandableRowContent>
                                            <VirtualMachineComponentsTable
                                                components={vulnerability.affectedComponents}
                                            />
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

export default VirtualMachineVulnerabilitiesTable;
