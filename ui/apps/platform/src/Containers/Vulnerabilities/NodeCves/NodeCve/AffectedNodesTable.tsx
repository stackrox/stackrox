import React from 'react';
import { Truncate, pluralize } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql } from '@apollo/client';
import { Link } from 'react-router-dom';

import { TableUIState } from 'utils/getTableUIState';

import useSet from 'hooks/useSet';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';

import CvssFormatted from 'Components/CvssFormatted';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import {
    getHighestVulnerabilitySeverity,
    getIsSomeVulnerabilityFixable,
    getHighestCvssScore,
} from '../../utils/vulnerabilityUtils';

import NodeComponentsTable from '../components/NodeComponentsTable';

export const affectedNodeFragment = gql`
    fragment AffectedNode on Node {
        id
        name
        operatingSystem
        cluster {
            name
        }
        nodeComponents {
            name
            version
            source
            nodeVulnerabilities {
                vulnerabilityId: id
                cve
                severity
                fixedByVersion
                cvss
                scoreVersion
            }
        }
    }
`;

export type AffectedNode = {
    id: string;
    name: string;
    operatingSystem: string;
    cluster: {
        name: string;
    };
    nodeComponents: {
        name: string;
        version: string;
        source: string;
        nodeVulnerabilities: {
            vulnerabilityId: string;
            cve: string;
            severity: string;
            fixedByVersion: string;
            cvss: number;
            scoreVersion: string;
        }[];
    }[];
};

export type AffectedNodesTableProps = {
    tableState: TableUIState<AffectedNode>;
};

// TODO Add filter icon to dynamic table columns
function AffectedNodesTable({ tableState }: AffectedNodesTableProps) {
    const colSpan = 8;
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
                    <Th>Node</Th>
                    <Th>CVE severity</Th>
                    <Th>CVE status</Th>
                    <Th>CVSS score</Th>
                    <Th>Cluster</Th>
                    <Th>Operating system</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{
                    message: 'There are no nodes that are affected by this CVE',
                }}
                renderer={({ data }) =>
                    data.map((node, rowIndex) => {
                        const { id, name, nodeComponents } = node;
                        const isExpanded = expandedRowSet.has(id);

                        const vulnerabilities = nodeComponents.flatMap(
                            (component) => component.nodeVulnerabilities
                        );
                        const topSeverity = getHighestVulnerabilitySeverity(vulnerabilities);
                        const isFixableInNode = getIsSomeVulnerabilityFixable(vulnerabilities);
                        const { cvss, scoreVersion } = getHighestCvssScore(vulnerabilities);

                        return (
                            <Tbody key={id} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(id),
                                        }}
                                    />

                                    <Td dataLabel="Node">
                                        <Link to={getNodeEntityPagePath('Node', id)}>
                                            <Truncate position="middle" content={name} />
                                        </Link>
                                    </Td>
                                    <Td dataLabel="CVE severity" modifier="nowrap">
                                        <VulnerabilitySeverityIconText severity={topSeverity} />
                                    </Td>
                                    <Td dataLabel="CVE status" modifier="nowrap">
                                        <VulnerabilityFixableIconText isFixable={isFixableInNode} />
                                    </Td>
                                    <Td dataLabel="CVSS score" modifier="nowrap">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    <Td dataLabel="Cluster">
                                        <Truncate position="middle" content={node.cluster.name} />
                                    </Td>
                                    <Td dataLabel="Operating system">{node.operatingSystem}</Td>
                                    <Td dataLabel="Affected components">
                                        {nodeComponents.length === 1
                                            ? nodeComponents[0].name
                                            : pluralize(nodeComponents.length, 'component')}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={colSpan - 1}>
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

export default AffectedNodesTable;
