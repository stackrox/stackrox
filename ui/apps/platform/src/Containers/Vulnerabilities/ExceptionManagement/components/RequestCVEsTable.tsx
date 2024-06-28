import React from 'react';
import { Bullseye, Flex, PageSection, Spinner, Text, Title } from '@patternfly/react-core';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';

import { SetResult } from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import {
    VulnerabilityExceptionScope,
    VulnerabilityState,
} from 'services/VulnerabilityExceptionService';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';

import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import {
    aggregateByCVSS,
    aggregateByCreatedTime,
    aggregateByDistinctCount,
    getScoreVersionsForTopCVSS,
    sortCveDistroList,
    getWorkloadSortFields,
    getDefaultWorkloadSortOption,
} from '../../utils/sortUtils';
import { CVEListQueryResult, cveListQuery } from '../../WorkloadCves/Tables/CVEsTable';
import { VulnerabilitySeverityLabel } from '../../types';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';

import { getImageScopeSearchValue } from '../utils';

type RequestCVEsTableProps = {
    cves: string[];
    scope: VulnerabilityExceptionScope;
    expandedRowSet: SetResult<string>;
    vulnerabilityState: VulnerabilityState;
};

function RequestCVEsTable({
    cves,
    scope,
    expandedRowSet,
    vulnerabilityState,
}: RequestCVEsTableProps) {
    const { page, perPage, setPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: getWorkloadSortFields('CVE'),
        defaultSortOption: getDefaultWorkloadSortOption('CVE'),
        onSort: () => setPage(1),
    });

    const queryObject = {
        CVE: cves.join(','),
        Image: getImageScopeSearchValue(scope),
    };

    const query = getRequestQueryStringForSearchFilter(queryObject);

    const { error, loading, data } = useQuery<CVEListQueryResult>(cveListQuery, {
        variables: {
            query,
            pagination: {
                offset: (page - 1) * perPage,
                limit: perPage,
                sortOption,
            },
        },
    });

    if (loading && !data) {
        return (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    }

    if (error) {
        return (
            <PageSection variant="light">
                <TableErrorComponent
                    error={error}
                    message="An error occurred. Try refreshing again"
                />
            </PageSection>
        );
    }

    return (
        <PageSection variant="light">
            <Flex direction={{ default: 'column' }}>
                <Title headingLevel="h2">{data?.imageCVEs.length || 0} results found</Title>
                <Table variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Td />
                            <Th sort={getSortParams('CVE')}>CVE</Th>
                            <Th>Images by severity</Th>
                            <Th sort={getSortParams('CVSS', aggregateByCVSS)}>CVSS</Th>
                            <Th sort={getSortParams('Image sha', aggregateByDistinctCount)}>
                                Affected images
                            </Th>
                            <Th sort={getSortParams('CVE Created Time', aggregateByCreatedTime)}>
                                First discovered
                            </Th>
                        </Tr>
                    </Thead>
                    {data?.imageCVEs.length === 0 && (
                        <Tbody>
                            <Tr>
                                <Td colSpan={6}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="No results found"
                                            headingLevel="h2"
                                            icon={SearchIcon}
                                        />
                                    </Bullseye>
                                </Td>
                            </Tr>
                        </Tbody>
                    )}
                    {data?.imageCVEs.length !== 0 &&
                        data?.imageCVEs.map(
                            (
                                {
                                    cve,
                                    affectedImageCountBySeverity,
                                    topCVSS,
                                    affectedImageCount,
                                    firstDiscoveredInSystem,
                                    distroTuples,
                                },
                                rowIndex
                            ) => {
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
                                        IMAGE: queryObject.Image,
                                    },
                                };
                                const cveURL = getWorkloadEntityPagePath(
                                    'CVE',
                                    cve,
                                    vulnerabilityState,
                                    cveURLQueryOptions
                                );

                                return (
                                    <Tbody key={cve}>
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
                                            <Td colSpan={5}>
                                                <ExpandableRowContent>
                                                    {prioritizedDistros.length > 0 && (
                                                        <Text>{summary}</Text>
                                                    )}
                                                </ExpandableRowContent>
                                            </Td>
                                        </Tr>
                                    </Tbody>
                                );
                            }
                        )}
                </Table>
            </Flex>
        </PageSection>
    );
}

export default RequestCVEsTable;
