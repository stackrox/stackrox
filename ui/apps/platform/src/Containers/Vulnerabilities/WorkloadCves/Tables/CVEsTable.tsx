import React from 'react';
import { gql } from '@apollo/client';
import {
    ActionsColumn,
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { Button, ButtonVariant, Text, pluralize } from '@patternfly/react-core';

import LinkShim from 'Components/PatternFly/LinkShim';
import { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import useMap from 'hooks/useMap';
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
import { ExceptionRequestModalOptions } from '../components/ExceptionRequestModal/ExceptionRequestModal';
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { CveSelectionsProps } from '../components/ExceptionRequestModal/CveSelections';

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

export type CVEsTableProps = {
    cves: ImageCVE[];
    unfilteredImageCount: number;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    showExceptionMenuItems: boolean;
    selectedCves: ReturnType<typeof useMap<string, CveSelectionsProps['cves'][number]>>;
    cveTableActionHandler: (opts: ExceptionRequestModalOptions) => void;
};

function CVEsTable({
    cves,
    unfilteredImageCount,
    getSortParams,
    isFiltered,
    filteredSeverities,
    showExceptionMenuItems,
    selectedCves,
    cveTableActionHandler,
}: CVEsTableProps) {
    const expandedRowSet = useSet<string>();

    const colSpan = 6 + (showExceptionMenuItems ? 2 : 0);

    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    {showExceptionMenuItems && (
                        <Th
                            title={
                                selectedCves.size > 0
                                    ? `Clear ${pluralize(selectedCves.size, 'selected CVE')}`
                                    : undefined
                            }
                            select={{
                                isSelected: selectedCves.size !== 0,
                                isDisabled: selectedCves.size === 0,
                                onSelect: selectedCves.clear,
                            }}
                        />
                    )}
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
                    {showExceptionMenuItems && <Th aria-label="CVE actions" />}
                </Tr>
            </Thead>
            {cves.length === 0 && <EmptyTableResults colSpan={colSpan} />}
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
                    const summary =
                        prioritizedDistros.length > 0 ? prioritizedDistros[0].summary : '';

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
                                {showExceptionMenuItems && (
                                    <Td
                                        key={cve}
                                        select={{
                                            rowIndex,
                                            onSelect: () => {
                                                if (selectedCves.has(cve)) {
                                                    selectedCves.remove(cve);
                                                } else {
                                                    selectedCves.set(cve, { cve, summary });
                                                }
                                            },
                                            isSelected: selectedCves.has(cve),
                                        }}
                                    />
                                )}
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
                                    {affectedImageCount}/{unfilteredImageCount} affected images
                                </Td>
                                <Td dataLabel="First discovered">
                                    <DateDistanceTd date={firstDiscoveredInSystem} />
                                </Td>
                                {showExceptionMenuItems && (
                                    <Td className="pf-u-px-0">
                                        <ActionsColumn
                                            items={[
                                                {
                                                    title: 'Defer CVE',
                                                    onClick: () =>
                                                        cveTableActionHandler({
                                                            type: 'DEFERRAL',
                                                            cves: [{ cve, summary }],
                                                        }),
                                                },
                                                {
                                                    title: 'Mark as false positive',
                                                    onClick: () =>
                                                        cveTableActionHandler({
                                                            type: 'FALSE_POSITIVE',
                                                            cves: [{ cve, summary }],
                                                        }),
                                                },
                                            ]}
                                        />
                                    </Td>
                                )}
                            </Tr>
                            <Tr isExpanded={isExpanded}>
                                <Td />
                                <Td colSpan={colSpan}>
                                    <ExpandableRowContent>
                                        {prioritizedDistros.length > 0 && <Text>{summary}</Text>}
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
