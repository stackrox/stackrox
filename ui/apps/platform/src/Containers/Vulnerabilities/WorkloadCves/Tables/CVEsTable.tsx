import React from 'react';
import { Link } from 'react-router-dom';
import { gql } from '@apollo/client';
import {
    ActionsColumn,
    ExpandableRowContent,
    IAction,
    Table,
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
import TooltipTh from 'Components/TooltipTh';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import { TableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import ExpandRowTh from 'Components/ExpandRowTh';
import { ACTION_COLUMN_POPPER_PROPS } from 'constants/tables';
import { VulnerabilitySeverityLabel } from '../../types';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import {
    getScoreVersionsForTopCVSS,
    sortCveDistroList,
    aggregateByCVSS,
    aggregateByCreatedTime,
    aggregateByDistinctCount,
} from '../../utils/sortUtils';
// eslint-disable-next-line @typescript-eslint/no-unused-vars
import { CveSelectionsProps } from '../../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';

export const cveListQuery = gql`
    query getImageCVEList(
        $query: String
        $pagination: Pagination
        $statusesForExceptionCount: [String!]
    ) {
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
            pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
        }
    }
`;

export const unfilteredImageCountQuery = gql`
    query getUnfilteredImageCount {
        imageCount
    }
`;

export type CVEListQueryResult = {
    imageCVEs: ImageCVE[];
};

export type ImageCVE = {
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
    pendingExceptionCount: number;
};

export type CVEsTableProps = {
    tableState: TableUIState<ImageCVE>;
    unfilteredImageCount: number;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    canSelectRows: boolean;
    selectedCves: ReturnType<typeof useMap<string, CveSelectionsProps['cves'][number]>>;
    createTableActions?: (cve: {
        cve: string;
        summary: string;
        numAffectedImages: number;
    }) => IAction[];
    vulnerabilityState: VulnerabilityState | undefined; // TODO Make Required when the ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL feature flag is removed
    onClearFilters: () => void;
};

function CVEsTable({
    tableState,
    unfilteredImageCount,
    getSortParams,
    isFiltered,
    filteredSeverities,
    canSelectRows,
    selectedCves,
    createTableActions,
    vulnerabilityState,
    onClearFilters,
}: CVEsTableProps) {
    const expandedRowSet = useSet<string>();
    const showExceptionDetailsLink = vulnerabilityState && vulnerabilityState !== 'OBSERVED';

    const colSpan =
        6 +
        (canSelectRows ? 1 : 0) +
        (createTableActions ? 1 : 0) +
        (showExceptionDetailsLink ? 1 : 0);

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
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
                        sort={getSortParams('Image sha', aggregateByDistinctCount)}
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
                    {createTableActions && (
                        <Th>
                            <span className="pf-v5-screen-reader">CVE actions</span>
                        </Th>
                    )}
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{ message: 'No CVEs have been observed in the system' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map(
                        (
                            {
                                cve,
                                affectedImageCountBySeverity,
                                topCVSS,
                                affectedImageCount,
                                firstDiscoveredInSystem,
                                distroTuples,
                                pendingExceptionCount,
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
                                        borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
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
                                                item={{
                                                    cve,
                                                    summary,
                                                    numAffectedImages: affectedImageCount,
                                                }}
                                            />
                                        )}
                                        <Td dataLabel="CVE" modifier="nowrap">
                                            <PendingExceptionLabelLayout
                                                hasPendingException={pendingExceptionCount > 0}
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            >
                                                <Link
                                                    to={getWorkloadEntityPagePath(
                                                        'CVE',
                                                        cve,
                                                        vulnerabilityState
                                                    )}
                                                >
                                                    {cve}
                                                </Link>
                                            </PendingExceptionLabelLayout>
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
                                            <CvssFormatted
                                                cvss={topCVSS}
                                                scoreVersion={
                                                    scoreVersions.length > 0
                                                        ? scoreVersions.join('/')
                                                        : undefined
                                                }
                                            />
                                        </Td>
                                        <Td dataLabel="Affected images">
                                            {affectedImageCount}/{unfilteredImageCount} affected
                                            images
                                        </Td>
                                        <Td dataLabel="First discovered">
                                            <DateDistance date={firstDiscoveredInSystem} />
                                        </Td>
                                        {showExceptionDetailsLink && (
                                            <ExceptionDetailsCell
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            />
                                        )}
                                        {createTableActions && (
                                            <Td className="pf-v5-u-px-0">
                                                <ActionsColumn
                                                    popperProps={ACTION_COLUMN_POPPER_PROPS}
                                                    items={createTableActions({
                                                        cve,
                                                        summary,
                                                        numAffectedImages: affectedImageCount,
                                                    })}
                                                />
                                            </Td>
                                        )}
                                    </Tr>
                                    <Tr isExpanded={isExpanded}>
                                        <Td />
                                        <Td colSpan={colSpan - 1}>
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
                        }
                    )
                }
            />
        </Table>
    );
}

export default CVEsTable;
