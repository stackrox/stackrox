import React from 'react';
import {
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
    ExpandableRowContent,
} from '@patternfly/react-table';
import { Button, ButtonVariant, Text } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import { graphql } from 'generated/graphql-codegen';
import { GetImageCveListQuery } from 'generated/graphql-codegen/graphql';
import { VulnerabilitySeverityLabel } from '../types';
import { getEntityPagePath } from '../searchUtils';
import TooltipTh from '../components/TooltipTh';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import DateDistanceTd from '../components/DatePhraseTd';
import CvssTd from '../components/CvssTd';
import {
    getScoreVersionsForTopCVSS,
    sortCveDistroList,
    aggregateByCVSS,
    aggregateByCreatedTime,
    aggregateByImageSha,
} from '../sortUtils';
import EmptyTableResults from '../components/EmptyTableResults';

export const cveListQuery = graphql(/* GraphQL */ `
    query getImageCVEList($query: String, $pagination: Pagination) {
        imageCVEs(query: $query, pagination: $pagination) {
            cve
            affectedImageCountBySeverity {
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
            affectedImageCount
            firstDiscoveredInSystem
            distroTuples {
                summary
                operatingSystem
                cvss
                scoreVersion
            }
        }
    }
`);

export const unfilteredImageCountQuery = graphql(/* GraphQL */ `
    query getUnfilteredImageCount {
        imageCount
    }
`);

type CVEsTableProps = {
    cves: GetImageCveListQuery['imageCVEs'];
    unfilteredImageCount: number;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
};

function CVEsTable({
    cves,
    unfilteredImageCount,
    getSortParams,
    isFiltered,
    filteredSeverities,
}: CVEsTableProps) {
    const expandedRowSet = useSet<string>();

    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <TooltipTh tooltip="Severity of this CVE across images">
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh
                        sort={getSortParams('CVSS', aggregateByCVSS)}
                        tooltip="Highest CVSS score of this CVE across images"
                    >
                        Top CVSS
                    </TooltipTh>
                    <TooltipTh
                        sort={getSortParams('Image sha', aggregateByImageSha)}
                        tooltip="Ratio of total images affected by this CVE"
                    >
                        Affected images
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh
                        sort={getSortParams('CVE Created Time', aggregateByCreatedTime)}
                        tooltip="Time since this CVE first affected an entity"
                    >
                        First discovered
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                </Tr>
            </Thead>
            {cves.length === 0 && <EmptyTableResults colSpan={6} />}
            {cves.map(
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

                    const prioritizedDistros = sortCveDistroList(distroTuples);
                    const scoreVersions = getScoreVersionsForTopCVSS(topCVSS, distroTuples);

                    return (
                        <Tbody
                            key={cve}
                            style={{
                                borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                            }}
                            isExpanded={isExpanded}
                        >
                            <Tr>
                                <Td
                                    expand={{
                                        rowIndex,
                                        isExpanded,
                                        onToggle: () => expandedRowSet.toggle(cve),
                                    }}
                                />
                                <Td dataLabel="CVE">
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('CVE', cve)}
                                    >
                                        {cve}
                                    </Button>
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
                                <Td dataLabel="Top CVSS">
                                    <CvssTd
                                        cvss={topCVSS}
                                        scoreVersion={
                                            scoreVersions.length > 0
                                                ? scoreVersions.join('/')
                                                : undefined
                                        }
                                    />
                                </Td>
                                <Td dataLabel="Affected images">
                                    {/* TODO: fix upon PM feedback */}
                                    {affectedImageCount}/{unfilteredImageCount} affected images
                                </Td>
                                <Td dataLabel="First discovered">
                                    <DateDistanceTd date={firstDiscoveredInSystem} />
                                </Td>
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={6}>
                                    <ExpandableRowContent>
                                        {prioritizedDistros.length > 0 && (
                                            <Text>{prioritizedDistros[0].summary}</Text>
                                        )}
                                    </ExpandableRowContent>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                }
            )}
        </TableComposable>
    );
}

export default CVEsTable;
