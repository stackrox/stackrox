import React from 'react';
import { Truncate, pluralize } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql } from '@apollo/client';
import { Link } from 'react-router-dom';

import { TableUIState } from 'utils/getTableUIState';

import useSet from 'hooks/useSet';
import { VulnerabilitySeverity, isVulnerabilitySeverity } from 'types/cve.proto';
import { severityRankings } from 'constants/vulnerabilities';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';

import CvssFormatted from 'Components/CvssFormatted';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import NodeComponentsTable from './NodeComponentsTable';

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

type NodeVulnerabilities = AffectedNode['nodeComponents'][0]['nodeVulnerabilities'];

function getHighestVulnerabilitySeverity(vulns: NodeVulnerabilities): VulnerabilitySeverity {
    let topSeverity: VulnerabilitySeverity = 'UNKNOWN_VULNERABILITY_SEVERITY';
    vulns.forEach(({ severity }) => {
        if (
            isVulnerabilitySeverity(severity) &&
            severityRankings[severity] > severityRankings[topSeverity]
        ) {
            topSeverity = severity;
        }
    });
    return topSeverity;
}

function getAnyVulnerabilityIsFixable(vulns: NodeVulnerabilities): boolean {
    return vulns.some((vuln) => vuln.fixedByVersion !== '');
}

function getHighestCvssScore(vulns: NodeVulnerabilities): {
    cvss: number;
    scoreVersion: string;
} {
    let topCvss = 0;
    let topScoreVersion = 'N/A';
    vulns.forEach(({ cvss, scoreVersion }) => {
        if (cvss > topCvss) {
            topCvss = cvss;
            topScoreVersion = scoreVersion;
        }
    });
    return { cvss: topCvss, scoreVersion: topScoreVersion };
}

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
                    message: 'No nodes are detected to have been reported for this CVE',
                }}
                renderer={({ data }) =>
                    data.map((node, rowIndex) => {
                        const { id, name, nodeComponents } = node;
                        const isExpanded = expandedRowSet.has(id);

                        const vulns = nodeComponents.flatMap(
                            ({ nodeVulnerabilities }) => nodeVulnerabilities
                        );
                        const topSeverity = getHighestVulnerabilitySeverity(vulns);
                        const isFixable = getAnyVulnerabilityIsFixable(vulns);
                        const { cvss, scoreVersion } = getHighestCvssScore(vulns);

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
                                        <VulnerabilityFixableIconText isFixable={isFixable} />
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
