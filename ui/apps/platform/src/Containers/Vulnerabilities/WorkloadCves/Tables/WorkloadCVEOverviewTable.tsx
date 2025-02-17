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

import useFeatureFlags from 'hooks/useFeatureFlags';
import { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import useMap from 'hooks/useMap';
import { CveBaseInfo, VulnerabilityState } from 'types/cve.proto';
import TooltipTh from 'Components/TooltipTh';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import { TableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import ExpandRowTh from 'Components/ExpandRowTh';
import { ACTION_COLUMN_POPPER_PROPS } from 'constants/tables';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';
import { VulnerabilitySeverityLabel } from '../../types';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import {
    getScoreVersionsForTopCVSS,
    getScoreVersionsForTopNvdCVSS,
    sortCveDistroList,
    aggregateByCVSS,
    aggregateByEPSS,
    aggregateByCreatedTime,
    aggregateByDistinctCount,
    getSeveritySortOptions,
} from '../../utils/sortUtils';
import { CveSelectionsProps } from '../../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { infoForEpssProbability } from './infoForTh';
import { formatEpssProbabilityAsPercent, getCveBaseInfoFromDistroTuples } from './table.utils';

export const tableId = 'WorkloadCveOverviewTable';
export const defaultColumns = {
    imagesBySeverity: {
        title: 'Images by severity',
        isShownByDefault: true,
    },
    topCvss: {
        title: 'Top CVSS',
        isShownByDefault: true,
    },
    topNvdCvss: {
        title: 'Top NVD CVSS',
        isShownByDefault: true,
    },
    epssProbability: {
        title: 'EPSS probability',
        isShownByDefault: true,
    },
    affectedImages: {
        title: 'Affected images',
        isShownByDefault: true,
    },
    firstDiscovered: {
        title: 'First discovered',
        isShownByDefault: true,
    },
    publishedOn: {
        title: 'Published',
        isShownByDefault: true,
    },
} as const;

export const cveListQuery = gql`
    query getImageCVEList(
        $query: String
        $pagination: Pagination
        $statusesForExceptionCount: [String!]
    ) {
        imageCVECount(query: $query)
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
            publishedOn
            topNvdCVSS
            distroTuples {
                summary
                operatingSystem
                cvss
                scoreVersion
                nvdCvss
                nvdScoreVersion
                cveBaseInfo {
                    epss {
                        epssProbability
                    }
                }
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
    imageCVECount: number;
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
    publishedOn: string | null;
    topNvdCVSS: number;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
        nvdCvss: number;
        nvdScoreVersion: string; // for example, V3 or UNKNOWN_VERSION
        cveBaseInfo: CveBaseInfo;
    }[];
    pendingExceptionCount: number;
};

export type WorkloadCVEOverviewTableProps = {
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
    vulnerabilityState: VulnerabilityState;
    onClearFilters: () => void;
    columnVisibilityState: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function WorkloadCVEOverviewTable({
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
    columnVisibilityState,
}: WorkloadCVEOverviewTableProps) {
    const { getAbsoluteUrl } = useWorkloadCveViewContext();
    const expandedRowSet = useSet<string>();
    const showExceptionDetailsLink = vulnerabilityState !== 'OBSERVED';
    const getVisibilityClass = generateVisibilityForColumns(columnVisibilityState);
    const hiddenColumnCount = getHiddenColumnCount(columnVisibilityState);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');

    const colSpan =
        6 +
        (isNvdCvssColumnEnabled ? 1 : 0) +
        (isEpssProbabilityColumnEnabled ? 1 : 0) +
        (canSelectRows ? 1 : 0) +
        (createTableActions ? 1 : 0) +
        (showExceptionDetailsLink ? 1 : 0) +
        -hiddenColumnCount;

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <TooltipTh
                        className={getVisibilityClass('imagesBySeverity')}
                        sort={getSortParams(
                            'Images By Severity',
                            getSeveritySortOptions(filteredSeverities)
                        )}
                        tooltip="Severity of this CVE across images"
                    >
                        Images by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh
                        className={getVisibilityClass('topCvss')}
                        sort={getSortParams('CVSS', aggregateByCVSS)}
                        tooltip="Highest CVSS score of this CVE across images"
                    >
                        Top CVSS
                    </TooltipTh>
                    {isNvdCvssColumnEnabled && (
                        <TooltipTh
                            className={getVisibilityClass('topNvdCvss')}
                            tooltip="Highest CVSS score (from National Vulnerability Database) of this CVE across images"
                        >
                            Top NVD CVSS
                        </TooltipTh>
                    )}
                    {isEpssProbabilityColumnEnabled && (
                        <Th
                            className={getVisibilityClass('epssProbability')}
                            info={infoForEpssProbability}
                            sort={getSortParams('EPSS Probability', aggregateByEPSS)}
                        >
                            EPSS probability
                        </Th>
                    )}
                    <TooltipTh
                        className={getVisibilityClass('affectedImages')}
                        sort={getSortParams('Image Sha', aggregateByDistinctCount)}
                        tooltip="Ratio of total images affected by this CVE"
                    >
                        Affected images
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh
                        className={getVisibilityClass('firstDiscovered')}
                        sort={getSortParams('CVE Created Time', aggregateByCreatedTime)}
                        tooltip="Time since this CVE first affected an entity"
                    >
                        First discovered
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <TooltipTh
                        className={getVisibilityClass('publishedOn')}
                        tooltip="Time when the CVE was made public and assigned a number"
                    >
                        Published
                    </TooltipTh>
                    {showExceptionDetailsLink && (
                        <TooltipTh tooltip="View information about this exception request">
                            Request details
                        </TooltipTh>
                    )}
                    {createTableActions && (
                        <Th>
                            <span className="pf-v5-screen-reader">Row actions</span>
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
                                topNvdCVSS,
                                affectedImageCount,
                                firstDiscoveredInSystem,
                                publishedOn,
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
                            const nvdScoreVersions = getScoreVersionsForTopNvdCVSS(
                                topNvdCVSS,
                                distroTuples
                            );
                            const cveBaseInfo = getCveBaseInfoFromDistroTuples(distroTuples);
                            const epssProbability = cveBaseInfo?.epss?.epssProbability;
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
                                                    to={getAbsoluteUrl(
                                                        getWorkloadEntityPagePath(
                                                            'CVE',
                                                            cve,
                                                            vulnerabilityState
                                                        )
                                                    )}
                                                >
                                                    {cve}
                                                </Link>
                                            </PendingExceptionLabelLayout>
                                        </Td>
                                        <Td
                                            dataLabel="Images by severity"
                                            className={getVisibilityClass('imagesBySeverity')}
                                        >
                                            <SeverityCountLabels
                                                criticalCount={criticalCount}
                                                importantCount={importantCount}
                                                moderateCount={moderateCount}
                                                lowCount={lowCount}
                                                filteredSeverities={filteredSeverities}
                                            />
                                        </Td>
                                        <Td
                                            dataLabel="Top CVSS"
                                            className={getVisibilityClass('topCvss')}
                                        >
                                            <CvssFormatted
                                                cvss={topCVSS}
                                                scoreVersion={
                                                    scoreVersions.length > 0
                                                        ? scoreVersions.join('/')
                                                        : undefined
                                                }
                                            />
                                        </Td>
                                        {isNvdCvssColumnEnabled && (
                                            <Td
                                                className={getVisibilityClass('topNvdCvss')}
                                                dataLabel="Top NVD CVSS"
                                            >
                                                <CvssFormatted
                                                    cvss={topNvdCVSS ?? 0}
                                                    scoreVersion={nvdScoreVersions.join('/')}
                                                />
                                            </Td>
                                        )}
                                        {isEpssProbabilityColumnEnabled && (
                                            <Td
                                                className={getVisibilityClass('epssProbability')}
                                                dataLabel="EPSS probability"
                                            >
                                                {formatEpssProbabilityAsPercent(epssProbability)}
                                            </Td>
                                        )}
                                        <Td
                                            dataLabel="Affected images"
                                            className={getVisibilityClass('affectedImages')}
                                        >
                                            {affectedImageCount}/{unfilteredImageCount} affected
                                            images
                                        </Td>
                                        <Td
                                            dataLabel="First discovered"
                                            className={getVisibilityClass('firstDiscovered')}
                                            modifier="nowrap"
                                        >
                                            <DateDistance date={firstDiscoveredInSystem} />
                                        </Td>
                                        <Td
                                            dataLabel="Published"
                                            className={getVisibilityClass('publishedOn')}
                                            modifier="nowrap"
                                        >
                                            {publishedOn ? (
                                                <DateDistance date={publishedOn} />
                                            ) : (
                                                'Not available'
                                            )}
                                        </Td>
                                        {showExceptionDetailsLink && (
                                            <ExceptionDetailsCell
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            />
                                        )}
                                        {createTableActions && (
                                            <Td isActionCell>
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

export default WorkloadCVEOverviewTable;
