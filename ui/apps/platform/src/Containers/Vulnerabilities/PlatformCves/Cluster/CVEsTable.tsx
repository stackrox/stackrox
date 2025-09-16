import React from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { gql } from '@apollo/client';
import { Text } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Td, ExpandableRowContent, Tbody } from '@patternfly/react-table';

import CvssFormatted from 'Components/CvssFormatted';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { TableUIState } from 'utils/getTableUIState';
import useSet from 'hooks/useSet';

import { UseURLSortResult } from 'hooks/useURLSort';
import ExpandRowTh from 'Components/ExpandRowTh';
import {
    CLUSTER_CVE_STATUS_SORT_FIELD,
    CVE_SORT_FIELD,
    CVE_TYPE_SORT_FIELD,
    CVSS_SORT_FIELD,
} from '../../utils/sortFields';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import { getPlatformEntityPagePath } from '../../utils/searchUtils';
import { displayCveType } from '../utils/stringUtils';

export const sortFields = [
    CVE_SORT_FIELD,
    CLUSTER_CVE_STATUS_SORT_FIELD,
    CVE_TYPE_SORT_FIELD,
    CVSS_SORT_FIELD,
];

export const defaultSortOption = { field: CVSS_SORT_FIELD, direction: 'desc' } as const;

export const clusterVulnerabilityFragment = gql`
    fragment ClusterVulnerabilityFragment on ClusterVulnerability {
        id
        cve
        isFixable
        cvss
        scoreVersion
        vulnerabilityType
        summary
    }
`;

export type ClusterVulnerability = {
    id: string;
    cve: string;
    isFixable: boolean;
    cvss: number;
    scoreVersion: string;
    vulnerabilityType: string;
    summary: string;
};

export type CVEsTableProps = {
    tableState: TableUIState<ClusterVulnerability>;
    getSortParams: UseURLSortResult['getSortParams'];
    onClearFilters: () => void;
};

function CVEsTable({ tableState, getSortParams, onClearFilters }: CVEsTableProps) {
    const COL_SPAN = 5;
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
                    <Th sort={getSortParams(CLUSTER_CVE_STATUS_SORT_FIELD)}>CVE status</Th>
                    <Th sort={getSortParams(CVE_TYPE_SORT_FIELD)}>CVE type</Th>
                    <Th sort={getSortParams(CVSS_SORT_FIELD)}>CVSS</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={COL_SPAN}
                emptyProps={{ message: 'No CVEs were detected for this cluster' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((clusterVulnerability, rowIndex) => {
                        const {
                            id,
                            cve,
                            isFixable,
                            vulnerabilityType,
                            cvss,
                            scoreVersion,
                            summary,
                        } = clusterVulnerability;
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
                                        <Link to={getPlatformEntityPagePath('CVE', id)}>{cve}</Link>
                                    </Td>
                                    <Td dataLabel="CVE status">
                                        <VulnerabilityFixableIconText isFixable={isFixable} />
                                    </Td>
                                    <Td dataLabel="CVE type">
                                        {displayCveType(vulnerabilityType)}
                                    </Td>
                                    <Td dataLabel="CVSS">
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={COL_SPAN - 1}>
                                        <ExpandableRowContent>
                                            {summary ? (
                                                <Text>{summary}</Text>
                                            ) : (
                                                <PartialCVEDataAlert />
                                            )}
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
