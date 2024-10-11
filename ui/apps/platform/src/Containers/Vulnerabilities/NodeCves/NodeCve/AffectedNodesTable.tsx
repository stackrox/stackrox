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
import { UseURLSortResult } from 'hooks/useURLSort';
import {
    CLUSTER_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
    NODE_SORT_FIELD,
    OPERATING_SYSTEM_SORT_FIELD,
} from '../../utils/sortFields';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import {
    getHighestVulnerabilitySeverity,
    getIsSomeVulnerabilityFixable,
    getHighestCvssScore,
} from '../../utils/vulnerabilityUtils';

import NodeComponentsTable, {
    NodeComponent,
    nodeComponentFragment,
} from '../components/NodeComponentsTable';

export const sortFields = [
    NODE_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
    CLUSTER_SORT_FIELD,
    OPERATING_SYSTEM_SORT_FIELD,
];

export const defaultSortOption = { field: CVE_SEVERITY_SORT_FIELD, direction: 'desc' } as const;

export const affectedNodeFragment = gql`
    ${nodeComponentFragment}
    fragment AffectedNode on Node {
        id
        name
        osImage
        cluster {
            name
        }
        nodeComponents(query: $query) {
            ...NodeComponentFragment
            nodeVulnerabilities(query: $query) {
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
    osImage: string;
    cluster: {
        name: string;
    };
    nodeComponents: (NodeComponent & {
        nodeVulnerabilities: {
            vulnerabilityId: string;
            cve: string;
            severity: string;
            fixedByVersion: string;
            cvss: number;
            scoreVersion: string;
        }[];
    })[];
};

export type AffectedNodesTableProps = {
    tableState: TableUIState<AffectedNode>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

// TODO Add filter icon to dynamic table columns
function AffectedNodesTable({
    tableState,
    getSortParams,
    onClearFilters,
}: AffectedNodesTableProps) {
    const colSpan = 8;
    const expandedRowSet = useSet<string>();

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            aria-live="polite"
            aria-busy={tableState.type === 'LOADING' ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th>
                        <span className="pf-v5-screen-reader">Row expansion</span>
                    </Th>
                    <Th sort={getSortParams(NODE_SORT_FIELD)}>Node</Th>
                    <Th sort={getSortParams(CVE_SEVERITY_SORT_FIELD)}>CVE severity</Th>
                    <Th sort={getSortParams(CVE_STATUS_SORT_FIELD)}>CVE status</Th>
                    <Th sort={getSortParams(CVSS_SORT_FIELD)}>CVSS</Th>
                    <Th sort={getSortParams(CLUSTER_SORT_FIELD)}>Cluster</Th>
                    <Th sort={getSortParams(OPERATING_SYSTEM_SORT_FIELD)}>Operating system</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{
                    message: 'There are no nodes that are affected by this CVE',
                }}
                filteredEmptyProps={{ onClearFilters }}
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
                                    <Td dataLabel="CVSS" modifier="nowrap">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    <Td dataLabel="Cluster">
                                        <Truncate position="middle" content={node.cluster.name} />
                                    </Td>
                                    <Td dataLabel="Operating system">
                                        <Truncate position="middle" content={node.osImage} />
                                    </Td>
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
                                            <NodeComponentsTable data={nodeComponents} />
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
