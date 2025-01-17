import React from 'react';
import {
    Flex,
    PageSection,
    Pagination,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { useQuery } from '@apollo/client';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import { SetResult } from 'hooks/useSet';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { VulnerabilityState } from 'services/VulnerabilityExceptionService';
import { getPaginationParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { getTableUIState } from 'utils/getTableUIState';
import { SearchFilter } from 'types/search';
import {
    aggregateByCVSS,
    aggregateByCreatedTime,
    aggregateByDistinctCount,
    getScoreVersionsForTopCVSS,
    sortCveDistroList,
    getWorkloadSortFields,
    getDefaultWorkloadSortOption,
    getSeveritySortOptions,
} from '../../utils/sortUtils';
import {
    CVEListQueryResult,
    cveListQuery,
} from '../../WorkloadCves/Tables/WorkloadCVEOverviewTable';
import { VulnerabilitySeverityLabel } from '../../types';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';

type RequestCVEsTableProps = {
    searchFilter: SearchFilter;
    vulnMgmtBaseUrl: string;
    pagination: UseURLPaginationResult;
    expandedRowSet: SetResult<string>;
    vulnerabilityState: VulnerabilityState;
};

function RequestCVEsTable({
    searchFilter,
    vulnMgmtBaseUrl,
    pagination,
    expandedRowSet,
    vulnerabilityState,
}: RequestCVEsTableProps) {
    const { page, perPage, setPage, setPerPage } = pagination;
    const { sortOption, getSortParams } = useURLSort({
        sortFields: getWorkloadSortFields('CVE'),
        defaultSortOption: getDefaultWorkloadSortOption('CVE'),
        onSort: () => setPage(1),
    });

    const query = getRequestQueryStringForSearchFilter(searchFilter);

    const {
        error,
        loading: isLoading,
        data,
    } = useQuery<CVEListQueryResult>(cveListQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        fetchPolicy: 'no-cache',
    });

    const tableState = getTableUIState({
        isLoading,
        data: data?.imageCVEs,
        error,
        searchFilter: {},
    });

    const colSpan = 6;

    return (
        <PageSection variant="light" className="pf-v5-u-pt-0">
            <Flex direction={{ default: 'column' }}>
                <Toolbar>
                    <ToolbarContent className="pf-v5-u-justify-content-space-between">
                        <ToolbarItem variant="label">
                            <Title headingLevel="h2">
                                {data?.imageCVECount || 0} results found
                            </Title>
                        </ToolbarItem>
                        <ToolbarItem variant="pagination">
                            <Pagination
                                itemCount={data?.imageCVECount}
                                perPage={perPage}
                                page={page}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    setPerPage(newPerPage);
                                }}
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <Table variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Th>
                                <span className="pf-v5-screen-reader">Row expansion</span>
                            </Th>
                            <Th sort={getSortParams('CVE')}>CVE</Th>
                            <Th
                                sort={getSortParams(
                                    'Images By Severity',
                                    getSeveritySortOptions([])
                                )}
                            >
                                Images by severity
                            </Th>
                            <Th sort={getSortParams('CVSS', aggregateByCVSS)}>CVSS</Th>
                            <Th sort={getSortParams('Image Sha', aggregateByDistinctCount)}>
                                Affected images
                            </Th>
                            <Th sort={getSortParams('CVE Created Time', aggregateByCreatedTime)}>
                                First discovered
                            </Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={colSpan}
                        emptyProps={{
                            title: 'No CVEs',
                            message: 'This request currently has no CVEs associated with it.',
                        }}
                        renderer={({ data }) =>
                            data.map((imageCVE, rowIndex) => {
                                const {
                                    cve,
                                    affectedImageCountBySeverity,
                                    topCVSS,
                                    affectedImageCount,
                                    firstDiscoveredInSystem,
                                    distroTuples,
                                } = imageCVE;
                                const isExpanded = expandedRowSet.has(cve);

                                const criticalCount = affectedImageCountBySeverity.critical.total;
                                const importantCount = affectedImageCountBySeverity.important.total;
                                const moderateCount = affectedImageCountBySeverity.moderate.total;
                                const lowCount = affectedImageCountBySeverity.low.total;
                                const filteredSeverities: VulnerabilitySeverityLabel[] = [
                                    'Critical',
                                    'Important',
                                    'Moderate',
                                    'Low',
                                ];
                                const prioritizedDistros = sortCveDistroList(distroTuples);
                                const scoreVersions = getScoreVersionsForTopCVSS(
                                    topCVSS,
                                    distroTuples
                                );
                                const summary =
                                    prioritizedDistros.length > 0
                                        ? prioritizedDistros[0].summary
                                        : '';

                                const cveURLQueryOptions = {
                                    s: {
                                        IMAGE: searchFilter.Image,
                                    },
                                };

                                const cveURL = `${vulnMgmtBaseUrl}/${getWorkloadEntityPagePath(
                                    'CVE',
                                    cve,
                                    vulnerabilityState,
                                    cveURLQueryOptions
                                )}`;

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
                                            <Td dataLabel="CVE">
                                                <Link to={cveURL}>{cve}</Link>
                                            </Td>
                                            <Td dataLabel="Images by severity">
                                                <SeverityCountLabels
                                                    criticalCount={criticalCount}
                                                    importantCount={importantCount}
                                                    moderateCount={moderateCount}
                                                    lowCount={lowCount}
                                                    filteredSeverities={filteredSeverities}
                                                />
                                            </Td>
                                            <Td dataLabel="CVSS">
                                                <CvssFormatted
                                                    cvss={topCVSS}
                                                    scoreVersion={
                                                        scoreVersions.length > 0
                                                            ? scoreVersions.join('/')
                                                            : undefined
                                                    }
                                                />
                                            </Td>
                                            <Td dataLabel="Affected images">{`${affectedImageCount} ${pluralize(
                                                'image',
                                                affectedImageCount
                                            )}`}</Td>
                                            <Td dataLabel="First discovered">
                                                <DateDistance date={firstDiscoveredInSystem} />
                                            </Td>
                                        </Tr>
                                        <Tr isExpanded={isExpanded}>
                                            <Td />
                                            <Td colSpan={colSpan - 1}>
                                                <ExpandableRowContent>
                                                    {prioritizedDistros.length > 0 && (
                                                        <Text>{summary}</Text>
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
            </Flex>
        </PageSection>
    );
}

export default RequestCVEsTable;
