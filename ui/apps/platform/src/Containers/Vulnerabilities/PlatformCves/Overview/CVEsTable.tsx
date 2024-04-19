import React from 'react';
import { Link } from 'react-router-dom';
import { Text } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td, ExpandableRowContent } from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';
import sum from 'lodash/sum';
import noop from 'lodash/noop';

import useURLPagination from 'hooks/useURLPagination';
import useSet from 'hooks/useSet';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import {
    TbodyLoading,
    TbodyError,
    TbodyEmpty,
    TbodyFilteredEmpty,
} from 'Components/TableStateTemplates';
import { getTableUIState } from 'utils/getTableUIState';

import TooltipTh from 'Components/TooltipTh';
import CvssFormatted from 'Components/CvssFormatted';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import PartialCVEDataAlert from '../../WorkloadCves/components/PartialCVEDataAlert';
import { sortCveDistroList } from '../../utils/sortUtils';
import { getPlatformEntityPagePath, getRegexScopedQueryString } from '../../utils/searchUtils';
import { QuerySearchFilter } from '../../types';

// TODO Validate types with BE implementation
type PlatformCVE = {
    cve: string;
    isFixable: boolean;
    cveType: string;
    cvss: number;
    scoreVersion: string;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
    }[];
    clusterCountByType: {
        generic: number;
        kubernetes: number;
        openshift: number;
        openshift4: number;
    };
};

const cveListQuery = gql`
    query getPlatformCves($query: String, $pagination: Pagination) {
        platformCVEs(query: $query, pagination: $pagination) {
            cve
            isFixable
            cveType
            cvss
            scoreVersion
            distroTuples {
                summary
                operatingSystem
            }
            clusterCountByType {
                generic
                kubernetes
                openshift
                openshift4
            }
        }
    }
`;

const totalClusterCountQuery = gql`
    query getTotalClusterCount {
        clusterCount
    }
`;

export type CVEsTableProps = {
    querySearchFilter: QuerySearchFilter;
    isFiltered: boolean;
    pagination: ReturnType<typeof useURLPagination>;
};

function CVEsTable({ querySearchFilter, isFiltered, pagination }: CVEsTableProps) {
    const { page, perPage } = pagination;

    const { error, loading, data, previousData } = useQuery<
        {
            platformCVEs: PlatformCVE[];
        },
        {
            query: string;
            pagination: {
                offset: number;
                limit: number;
            };
        }
    >(cveListQuery, {
        variables: {
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
            },
        },
    });

    const totalClusterCountRequest = useQuery(totalClusterCountQuery);
    const totalClusterCount = totalClusterCountRequest.data?.clusterCount ?? 0;

    const tableData = data ?? previousData;

    const tableState = getTableUIState({
        isLoading: loading,
        data: tableData?.platformCVEs,
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
                    <Th>CVE status</Th>
                    <Th>CVE type</Th>
                    <Th>CVSS</Th>
                    <TooltipTh tooltip="Ratio of the number of clusters affected by this CVE to the total number of secured clusters">
                        Affected clusters
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
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
            {tableState.type === 'FILTERED_EMPTY' && (
                <TbodyFilteredEmpty colSpan={colSpan} onClearFilters={noop} />
            )}
            {tableState.type === 'COMPLETE' &&
                tableState.data.map((platformCve, rowIndex) => {
                    const {
                        cve,
                        isFixable,
                        cveType,
                        cvss,
                        scoreVersion,
                        distroTuples,
                        clusterCountByType,
                    } = platformCve;
                    const isExpanded = expandedRowSet.has(cve);

                    const prioritizedDistros = sortCveDistroList(distroTuples);
                    const summary =
                        prioritizedDistros.length > 0 ? prioritizedDistros[0].summary : '';
                    const affectedClusterCount = sum(Object.values(clusterCountByType));

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
                                    <Link to={getPlatformEntityPagePath('CVE', cve)}>{cve}</Link>
                                </Td>
                                <Td dataLabel="CVE status">
                                    <VulnerabilityFixableIconText isFixable={isFixable} />
                                </Td>
                                <Td dataLabel="CVE type">{cveType}</Td>
                                <Td dataLabel="CVSS">
                                    <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                </Td>
                                <Td dataLabel="Affected clusters">
                                    {affectedClusterCount} / {totalClusterCount} affected clusters
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
