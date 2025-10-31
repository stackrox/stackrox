import { Link } from 'react-router-dom-v5-compat';
import { gql } from '@apollo/client';
import { pluralize } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import type { TableUIState } from 'utils/getTableUIState';
import useSet from 'hooks/useSet';

import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import type { UseURLSortResult } from 'hooks/useURLSort';
import ExpandRowTh from 'Components/ExpandRowTh';
import {
    getHighestVulnerabilitySeverity,
    getIsSomeVulnerabilityFixable,
} from '../../utils/vulnerabilityUtils';
import { getNodeEntityPagePath } from '../../utils/searchUtils';
import {
    CVE_SEVERITY_SORT_FIELD,
    CVE_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import NodeComponentsTable, { nodeComponentFragment } from '../components/NodeComponentsTable';
import type { NodeComponent } from '../components/NodeComponentsTable';

export const sortFields = [
    CVE_SORT_FIELD,
    CVE_SEVERITY_SORT_FIELD,
    CVE_STATUS_SORT_FIELD,
    CVSS_SORT_FIELD,
    // TODO - Needs a BE field implementation
    //  AFFECTED_COMPONENTS_SORT_FIELD
];

export const defaultSortOption = { field: CVE_SEVERITY_SORT_FIELD, direction: 'desc' } as const;

export const nodeVulnerabilityFragment = gql`
    ${nodeComponentFragment}
    fragment NodeVulnerabilityFragment on NodeVulnerability {
        cve
        summary
        cvss
        scoreVersion
        nodeComponents(query: $query) {
            ...NodeComponentFragment
        }
    }
`;

export type NodeVulnerability = {
    cve: string;
    summary: string;
    cvss: number;
    scoreVersion: string;
    nodeComponents: NodeComponent[];
};

export type CVEsTableProps = {
    tableState: TableUIState<NodeVulnerability>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function CVEsTable({ tableState, getSortParams, onClearFilters }: CVEsTableProps) {
    const COL_SPAN = 6;
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
                    <ExpandRowTh />
                    <Th sort={getSortParams(CVE_SORT_FIELD)}>CVE</Th>
                    <Th sort={getSortParams(CVE_SEVERITY_SORT_FIELD)}>Top severity</Th>
                    <Th sort={getSortParams(CVE_STATUS_SORT_FIELD)}>CVE status</Th>
                    <Th sort={getSortParams(CVSS_SORT_FIELD)}>CVSS</Th>
                    <Th>Affected components</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={COL_SPAN}
                emptyProps={{ message: 'No CVEs were detected for this node' }}
                filteredEmptyProps={{ onClearFilters }}
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
                                    <Td dataLabel="CVSS">
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

export default CVEsTable;
