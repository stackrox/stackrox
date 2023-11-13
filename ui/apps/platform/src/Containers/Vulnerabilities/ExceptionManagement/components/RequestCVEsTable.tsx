import React from 'react';
import { Bullseye, Flex, PageSection, Spinner, Text, Title } from '@patternfly/react-core';
import {
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import { Link } from 'react-router-dom';
import pluralize from 'pluralize';
import omitBy from 'lodash/omitBy';

import { getEntityPagePath } from 'Containers/Vulnerabilities/WorkloadCves/searchUtils';
import { VulnerabilitySeverityLabel } from 'Containers/Vulnerabilities/WorkloadCves/types';
import {
    CVEListQueryResult,
    cveListQuery,
} from 'Containers/Vulnerabilities/WorkloadCves/Tables/CVEsTable';
import { SetResult } from 'hooks/useSet';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import {
    CVEsDefaultSort,
    aggregateByCVSS,
    aggregateByCreatedTime,
    aggregateByImageSha,
    defaultCVESortFields,
    getScoreVersionsForTopCVSS,
    sortCveDistroList,
} from 'Containers/Vulnerabilities/WorkloadCves/sortUtils';
import { VulnerabilityExceptionScope } from 'services/VulnerabilityExceptionService';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
// @TODO: Move these files up to a common directory and move the types used in these files as well
import TableErrorComponent from 'Containers/Vulnerabilities/WorkloadCves/components/TableErrorComponent';
import SeverityCountLabels from '../../WorkloadCves/components/SeverityCountLabels';
import CvssTd from '../../WorkloadCves/components/CvssTd';
import DateDistanceTd from '../../WorkloadCves/components/DatePhraseTd';

type RequestCVEsTableProps = {
    cves: string[];
    scope: VulnerabilityExceptionScope;
    expandedRowSet: SetResult<string>;
};

function getImageScope(scope: VulnerabilityExceptionScope): string {
    if (
        scope.imageScope.registry === '.*' &&
        scope.imageScope.remote === '.*' &&
        scope.imageScope.tag === '.*'
    ) {
        return '';
    }
    if (scope.imageScope.tag === '.*') {
        return `${scope.imageScope.registry}/${scope.imageScope.remote}`;
    }
    return `${scope.imageScope.registry}/${scope.imageScope.remote}:${scope.imageScope.tag}`;
}

function RequestCVEsTable({ cves, scope, expandedRowSet }: RequestCVEsTableProps) {
    const { page, perPage, setPage } = useURLPagination(20);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultCVESortFields,
        defaultSortOption: CVEsDefaultSort,
        onSort: () => setPage(1),
    });

    const queryObject = {
        CVE: cves.join(','),
        Image: getImageScope(scope),
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
                <Spinner isSVG />
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
                <TableComposable variant="compact">
                    <Thead noWrap>
                        <Tr>
                            <Td />
                            <Th sort={getSortParams('CVE')}>CVE</Th>
                            <Th>Images by severity</Th>
                            <Th sort={getSortParams('CVSS', aggregateByCVSS)}>CVSS</Th>
                            <Th sort={getSortParams('Image sha', aggregateByImageSha)}>
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
                                    vulnerabilityState: 'DEFERRED',
                                };
                                // @TODO: This needs to be tested more thoroughly. Once the deferred tab shows the correct data, we should add a test for this
                                const cveURL = getEntityPagePath(
                                    'CVE',
                                    cve,
                                    omitBy(cveURLQueryOptions, (value) => value === '')
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
                                            <Td>
                                                <Link to={cveURL}>{cve}</Link>
                                            </Td>
                                            <Td>
                                                <SeverityCountLabels
                                                    criticalCount={criticalCount}
                                                    importantCount={importantCount}
                                                    moderateCount={moderateCount}
                                                    lowCount={lowCount}
                                                    filteredSeverities={filteredSeverities}
                                                />
                                            </Td>
                                            <Td>
                                                <CvssTd
                                                    cvss={topCVSS}
                                                    scoreVersion={
                                                        scoreVersions.length > 0
                                                            ? scoreVersions.join('/')
                                                            : undefined
                                                    }
                                                />
                                            </Td>
                                            <Td>{`${affectedImageCount} ${pluralize(
                                                'image',
                                                affectedImageCount
                                            )}`}</Td>
                                            <Td>
                                                <DateDistanceTd date={firstDiscoveredInSystem} />
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
                </TableComposable>
            </Flex>
        </PageSection>
    );
}

export default RequestCVEsTable;
