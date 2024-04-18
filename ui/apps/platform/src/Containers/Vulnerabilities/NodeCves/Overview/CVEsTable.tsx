import React from 'react';
import { Link } from 'react-router-dom';
import { Text } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';

import {
    TbodyLoading,
    TbodyError,
    TbodyEmpty,
    TbodyFilteredEmpty,
} from 'Components/TableStateTemplates';
import useSet from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import { getTableUIState } from 'utils/getTableUIState';

import TooltipTh from 'Components/TooltipTh';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from '../../../../Components/DateDistance';
import { getNodeEntityPagePath, getRegexScopedQueryString } from '../../utils/searchUtils';
import PartialCVEDataAlert from '../../WorkloadCves/components/PartialCVEDataAlert';
import { getScoreVersionsForTopCVSS, sortCveDistroList } from '../../utils/sortUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { QuerySearchFilter } from '../../types';

const cvesListQuery = gql`
    query getNodeCVEs($query: String, $pagination: Pagination) {
        nodeCVEs(query: $query, pagination: $pagination) {
            cve
            nodeCountBySeverity {
                critical {
                    total
                }
                important {
                    total
                }
                moderate {
                    total
                }
                low {
                    total
                }
            }
            topCVSS
            affectedNodeCount
            firstDiscoveredInSystem
            distroTuples {
                summary
                operatingSystem
                cvss
                scoreVersion
            }
        }
    }
`;

const totalNodeCountQuery = gql`
    query getTotalNodeCount {
        nodeCount
    }
`;

// TODO Need to verify these types with the BE implementation
export type NodeCVE = {
    cve: string;
    nodeCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    topCVSS: number;
    affectedNodeCount: number;
    firstDiscoveredInSystem: string;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
    }[];
};

export type CVEsTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
};

function CVEsTable({ querySearchFilter, isFiltered, pagination }: CVEsTableProps) {
    const { page, perPage } = pagination;
    const { data, previousData, loading, error } = useQuery<
        { nodeCVEs: NodeCVE[] },
        {
            query: string;
            pagination: {
                offset: number;
                limit: number;
            };
        }
    >(cvesListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
            },
        },
    });

    const totalNodeCountRequest = useQuery<{ nodeCount: number }>(totalNodeCountQuery);
    const totalNodeCount = totalNodeCountRequest.data?.nodeCount ?? 0;

    const tableData = data ?? previousData;
    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.nodeCVEs,
        error,
        searchFilter: querySearchFilter,
    });

    const expandedRowSet = useSet<string>();
    const colSpan = 6;

    return (
        <Table
            borders={tableState.type === 'COMPLETE'}
            variant="compact"
            role="region"
            aria-live="polite"
            aria-busy={loading ? 'true' : 'false'}
        >
            <Thead noWrap>
                <Tr>
                    <Th aria-label="Expand row" />
                    <Th>CVE</Th>
                    <TooltipTh tooltip="The number of nodes affected by this CVE, grouped by the severity of the CVE on each node">
                        Nodes by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th>Top CVSS</Th>
                    <TooltipTh tooltip="Ratio of the number of nodes affected by this CVE to the total number of nodes">
                        Affected nodes
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th>First discovered</Th>
                </Tr>
            </Thead>
            {tableState.type === 'LOADING' && <TbodyLoading colSpan={colSpan} />}
            {tableState.type === 'ERROR' && (
                <TbodyError colSpan={colSpan} error={tableState.error} />
            )}
            {tableState.type === 'EMPTY' && (
                <TbodyEmpty
                    colSpan={colSpan}
                    message="No Platform CVEs have been reported for your secured clusters"
                />
            )}
            {tableState.type === 'FILTERED_EMPTY' && <TbodyFilteredEmpty colSpan={colSpan} />}
            {tableState.type === 'COMPLETE' &&
                tableState.data.map((nodeCve, rowIndex) => {
                    const {
                        cve,
                        nodeCountBySeverity: { critical, important, moderate, low },
                        distroTuples,
                        topCVSS,
                        affectedNodeCount,
                        firstDiscoveredInSystem,
                    } = nodeCve;
                    const isExpanded = expandedRowSet.has(cve);

                    const prioritizedDistros = sortCveDistroList(distroTuples);
                    const summary =
                        prioritizedDistros.length > 0 ? prioritizedDistros[0].summary : '';
                    const scoreVersions = getScoreVersionsForTopCVSS(topCVSS, distroTuples);

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
                                <Td dataLabel="Nodes by severity">
                                    <SeverityCountLabels
                                        criticalCount={critical.total}
                                        importantCount={important.total}
                                        moderateCount={moderate.total}
                                        lowCount={low.total}
                                        // TODO - Add filtered severities once filter toolbar is in place
                                    />
                                </Td>
                                <Td dataLabel="Top CVSS">
                                    <CvssFormatted
                                        cvss={topCVSS}
                                        scoreVersion={
                                            scoreVersions.length > 0
                                                ? scoreVersions.join('/')
                                                : undefined
                                        }
                                    />
                                </Td>
                                <Td dataLabel="Affected nodes">
                                    {affectedNodeCount} / {totalNodeCount} affected nodes
                                </Td>
                                <Td dataLabel="First discovered">
                                    <DateDistance date={firstDiscoveredInSystem} />
                                </Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={colSpan - 1}>
                                    <ExpandableRowContent>
                                        {summary ? <Text>{summary}</Text> : <PartialCVEDataAlert />}
                                    </ExpandableRowContent>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                })}
        </Table>
    );
}

export default CVEsTable;
