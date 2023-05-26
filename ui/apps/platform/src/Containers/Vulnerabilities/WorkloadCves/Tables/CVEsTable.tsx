import React from 'react';
import { gql } from '@apollo/client';
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
import { VulnerabilitySeverityLabel } from '../types';
import { getEntityPagePath } from '../searchUtils';
import TooltipTh from '../components/TooltipTh';
import SeverityCountLabels from '../components/SeverityCountLabels';
import { DynamicColumnIcon } from '../components/DynamicIcon';
import DatePhraseTd from '../components/DatePhraseTd';
import CvssTd from '../components/CvssTd';
import { getScoreVersionsForTopCVSS, sortCveDistroList } from '../sortUtils';

export const cveListQuery = gql`
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
`;

export const unfilteredImageCountQuery = gql`
    query getUnfilteredImageCount {
        imageCount
    }
`;

type ImageCVE = {
    cve: string;
    affectedImageCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
    };
    topCVSS: number;
    affectedImageCount: number;
    firstDiscoveredInSystem: string | null;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
    }[];
};

type CVEsTableProps = {
    cves: ImageCVE[];
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
                {/* TODO: need to double check sorting on columns  */}
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <TooltipTh tooltip="Severity of this CVE across images">
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh tooltip="Highest CVSS score of this CVE across images">
                        Top CVSS
                    </TooltipTh>
                    <TooltipTh tooltip="Ratio of total environment affect by this CVE">
                        Affected images
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh tooltip="Time since this CVE first affected an entity">
                        First discovered
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                </Tr>
            </Thead>
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
                                <Td>
                                    <Button
                                        variant={ButtonVariant.link}
                                        isInline
                                        component={LinkShim}
                                        href={getEntityPagePath('CVE', cve)}
                                    >
                                        {cve}
                                    </Button>
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
                                <Td>
                                    {/* TODO: fix upon PM feedback */}
                                    {affectedImageCount}/{unfilteredImageCount} affected images
                                </Td>
                                <Td>
                                    <DatePhraseTd date={firstDiscoveredInSystem} />
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
