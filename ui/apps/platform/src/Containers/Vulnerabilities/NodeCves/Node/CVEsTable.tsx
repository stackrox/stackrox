import React from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';
import { pluralize } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Td, ExpandableRowContent, Tbody } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { TableUIState } from 'utils/getTableUIState';
import useSet from 'hooks/useSet';

import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import {
    getIsSomeVulnerabilityFixable,
    getHighestVulnerabilitySeverity,
} from '../../utils/vulnerabilityUtils';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import NodeComponentsTable from '../components/NodeComponentsTable';

export const nodeVulnerabilityFragment = gql`
    fragment NodeVulnerabilityFragment on NodeVulnerability {
        cve
        summary
        cvss
        scoreVersion
        nodeComponents(query: $query) {
            name
            source
            operatingSystem
            version
            nodeVulnerabilities(query: $query) {
                severity
                isFixable
                fixedByVersion
            }
        }
    }
`;

export type NodeVulnerability = {
    cve: string;
    summary: string;
    cvss: number;
    scoreVersion: string;
    nodeComponents: {
        name: string;
        source: string;
        operatingSystem: string;
        version: string;
        nodeVulnerabilities: {
            severity: string;
            isFixable: boolean;
            fixedByVersion: string;
        }[];
    }[];
};

export type CVEsTableProps = {
    tableState: TableUIState<NodeVulnerability>;
};

function CVEsTable({ tableState }: CVEsTableProps) {
    const COL_SPAN = 6;
    const expandedRowSet = useSet<string>();

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            role="region"
            aria-live="polite"
            aria-busy={tableState.type === 'LOADING' ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th aria-label="Expand row" />
                    <Th>CVE</Th>
                    <Th>Top severity</Th>
                    <Th>CVE status</Th>
                    <Th>CVSS score</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={COL_SPAN}
                emptyProps={{ message: 'No CVEs were detected for this node' }}
                renderer={({ data }) =>
                    data.map((nodeVulnerability, rowIndex) => {
                        const { cve, cvss, scoreVersion, nodeComponents } = nodeVulnerability;

                        const vulnerabilities = nodeComponents.flatMap(
                            (component) => component.nodeVulnerabilities
                        );
                        const topSeverity = getHighestVulnerabilitySeverity(vulnerabilities);
                        const isFixableInNode = getIsSomeVulnerabilityFixable(vulnerabilities);
                        const isExpanded = expandedRowSet.has(cve);

                        return (
                            <Tbody key={cve} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(cve),
                                        }}
                                    />
                                    <Td dataLabel="CVE" modifier="nowrap">
                                        <Link to={getNodeEntityPagePath('CVE', cve)}>{cve}</Link>
                                    </Td>
                                    <Td dataLabel="Top severity">
                                        <VulnerabilitySeverityIconText severity={topSeverity} />
                                    </Td>
                                    <Td dataLabel="CVE status">
                                        <VulnerabilityFixableIconText isFixable={isFixableInNode} />
                                    </Td>
                                    <Td dataLabel="CVSS score">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    <Td dataLabel="Affected components">
                                        {nodeComponents.length === 1
                                            ? nodeComponents[0].name
                                            : pluralize(nodeComponents.length, 'component')}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={COL_SPAN - 1}>
                                        <ExpandableRowContent>
                                            <NodeComponentsTable />
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

export default CVEsTable;
