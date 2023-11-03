import React from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import { Text } from '@patternfly/react-core';

import { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import useMap from 'hooks/useMap';
import { VulnerabilityState } from 'types/cve.proto';
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
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { CveSelectionsProps } from '../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../components/CVESelectionTh';
import CVESelectionTd from '../components/CVESelectionTd';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';

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
    canSelectRows: boolean;
    selectedCves: ReturnType<typeof useMap<string, CveSelectionsProps['cves'][number]>>;
    createTableActions?: (cve: { cve: string; summary: string }) => IAction[];
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make Required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
};

function CVEsTable({
    cves,
    unfilteredImageCount,
    getSortParams,
    isFiltered,
    filteredSeverities,
    canSelectRows,
    selectedCves,
    createTableActions,
    vulnerabilityState,
}: CVEsTableProps) {
    const expandedRowSet = useSet<string>();
    const showExceptionDetailsLink = vulnerabilityState && vulnerabilityState !== 'OBSERVED';

    const colSpan =
        6 +
        (canSelectRows ? 1 : 0) +
        (createTableActions ? 1 : 0) +
        (showExceptionDetailsLink ? 1 : 0);

    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th>{/* Header for expanded column */}</Th>
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
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
                    {showExceptionDetailsLink && (
                        <TooltipTh tooltip="View information about this exception request">
                            Request details
                        </TooltipTh>
                    )}
                    {createTableActions && <Th aria-label="CVE actions" />}
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
                                {canSelectRows && (
                                    <CVESelectionTd
                                        selectedCves={selectedCves}
                                        rowIndex={rowIndex}
                                        cve={cve}
                                        summary={summary}
                                    />
                                )}
                                <Td dataLabel="CVE">
                                    <Link to={getEntityPagePath('CVE', cve)}>{cve}</Link>
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
                                {showExceptionDetailsLink && (
                                    <ExceptionDetailsCell
                                        cve={cve}
                                        vulnerabilityState={vulnerabilityState}
                                    />
                                )}
                                {createTableActions && (
                                    <Td className="pf-u-px-0">
                                        <ActionsColumn
                                            items={createTableActions({ cve, summary })}
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
