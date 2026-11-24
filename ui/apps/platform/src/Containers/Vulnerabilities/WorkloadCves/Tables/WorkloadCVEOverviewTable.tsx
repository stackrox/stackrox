import type { ReactNode } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { gql } from '@apollo/client';
import {
    ActionsColumn,
    ExpandableRowContent,
    Table,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';
import type { IAction } from '@patternfly/react-table';
import { LabelGroup } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import type { UseURLSortResult } from 'hooks/useURLSort';
import useSet from 'hooks/useSet';
import type useMap from 'hooks/useMap';
import type { CveBaseInfo, VulnerabilityState } from 'types/cve.proto';
import TooltipTh from 'Components/TooltipTh';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import type { TableUIState } from 'utils/getTableUIState';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import ExpandRowTh from 'Components/ExpandRowTh';
import { ACTION_COLUMN_POPPER_PROPS } from 'constants/tables';
import { generateVisibilityForColumns, getHiddenColumnCount } from 'hooks/useManagedColumns';
import type { ManagedColumns } from 'hooks/useManagedColumns';
import {
    aggregateByCVSS,
    aggregateByCreatedTime,
    aggregateByDistinctCount,
    aggregateByEPSS,
} from '../../utils/sortUtils';
import type { CveSelectionsProps } from '../../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import KnownExploitLabel from '../../components/KnownExploitLabel';
import KnownRansomwareCampaignLabel from '../../components/KnownRansomwareCampaignLabel';
import PendingExceptionLabel from '../../components/PendingExceptionLabel';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';
import CVESummaryContent from '../components/CVESummaryContent';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { infoForEpssProbability } from './infoForTh';
import { formatEpssProbabilityAsPercent } from './table.utils';

export const tableId = 'WorkloadCveOverviewTable';
export const defaultColumns = {
    rowExpansion: {
        title: 'Row expansion',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cveSelection: {
        title: 'CVE selection',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cve: {
        title: 'CVE',
        isShownByDefault: true,
        isUntoggleAble: true,
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
    requestDetails: {
        title: 'Request details',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    rowActions: {
        title: 'Row actions',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
} as const;

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
                unknown {
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
                    exploit {
                        knownRansomwareCampaignUse
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
    imageCVEs: ImageCVE[];
};

export type ImageCVE = {
    cve: string;
    affectedImageCountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
        unknown: { total: number };
    };
    topCVSS: number | string;
    affectedImageCount: number;
    firstDiscoveredInSystem: string | null;
    publishedOn: string | null;
    topNvdCVSS: number | null;
    distroTuples: {
        summary: string;
        operatingSystem: string;
        cvss: number;
        scoreVersion: string;
        nvdCvss: number | null;
        nvdScoreVersion: string; // for example, V3 or UNKNOWN_VERSION
        cveBaseInfo: CveBaseInfo;
    }[];
    pendingExceptionCount: number;
};

// Shape returned by the vulnerability management REST API for the CVE list.
export type WorkloadCVEListItem = {
    cve: string;
    topCVSS: number;
    topNvdCVSS: number;
    topEpssProbability: number;
    affectedImageCount: number;
    firstDiscoveredInSystem: string | null;
    publishedOn: string | null;
    pendingExceptionCount: number;
    knownExploit: boolean;
    knownRansomwareCampaignUse: boolean;
};

export type WorkloadCVEOverviewTableProps = {
    tableState: TableUIState<WorkloadCVEListItem>;
    unfilteredImageCount: number;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    selectedCves: ReturnType<typeof useMap<string, CveSelectionsProps['cves'][number]>>;
    createTableActions?: (cve: {
        cve: string;
        summary: string;
        numAffectedImages: number;
    }) => IAction[];
    vulnerabilityState: VulnerabilityState;
    onClearFilters: () => void;
    columnVisibilityState: ManagedColumns<keyof typeof defaultColumns>['columns'];
    queryString: string;
};

function WorkloadCVEOverviewTable({
    tableState,
    unfilteredImageCount,
    getSortParams,
    isFiltered,
    selectedCves,
    createTableActions = () => [],
    vulnerabilityState,
    onClearFilters,
    columnVisibilityState,
    queryString,
}: WorkloadCVEOverviewTableProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { urlBuilder } = useWorkloadCveViewContext();
    const expandedRowSet = useSet<string>();
    const getVisibilityClass = generateVisibilityForColumns(columnVisibilityState);
    const hiddenColumnCount = getHiddenColumnCount(columnVisibilityState);
    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <CVESelectionTh
                        className={getVisibilityClass('cveSelection')}
                        selectedCves={selectedCves}
                    />
                    <Th sort={getSortParams('CVE')}>CVE</Th>
                    <TooltipTh
                        className={getVisibilityClass('topCvss')}
                        sort={getSortParams('CVSS', aggregateByCVSS)}
                        tooltip="Highest CVSS score of this CVE across images"
                    >
                        Top CVSS
                    </TooltipTh>
                    <TooltipTh
                        className={getVisibilityClass('topNvdCvss')}
                        tooltip="Highest CVSS score (from National Vulnerability Database) of this CVE across images"
                    >
                        Top NVD CVSS
                    </TooltipTh>
                    <Th
                        className={getVisibilityClass('epssProbability')}
                        info={infoForEpssProbability}
                        sort={getSortParams('EPSS Probability', aggregateByEPSS)}
                    >
                        EPSS probability
                    </Th>
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
                    <TooltipTh
                        className={getVisibilityClass('requestDetails')}
                        tooltip="View information about this exception request"
                    >
                        Request details
                    </TooltipTh>
                    <Th
                        className={getVisibilityClass('rowActions')}
                        screenReaderText="Row actions"
                    />
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
                                topCVSS,
                                topNvdCVSS,
                                topEpssProbability,
                                affectedImageCount,
                                firstDiscoveredInSystem,
                                publishedOn,
                                pendingExceptionCount,
                                knownExploit,
                                knownRansomwareCampaignUse,
                            },
                            rowIndex
                        ) => {
                            const isExpanded = expandedRowSet.has(cve);
                            // The summary is loaded lazily on expansion; actions receive an
                            // empty summary until then.
                            const summary = '';

                            const labels: ReactNode[] = [];
                            if (
                                isFeatureFlagEnabled('ROX_SCANNER_V4') &&
                                knownExploit
                            ) {
                                labels.push(<KnownExploitLabel key="exploit" isCompact />);
                                if (knownRansomwareCampaignUse) {
                                    labels.push(
                                        <KnownRansomwareCampaignLabel
                                            key="knownRansomwareCampaignUse"
                                            isCompact
                                        />
                                    );
                                }
                            }
                            if (pendingExceptionCount > 0) {
                                labels.push(
                                    <PendingExceptionLabel
                                        key="pendingExceptionCount"
                                        cve={cve}
                                        isCompact
                                        vulnerabilityState={vulnerabilityState}
                                    />
                                );
                            }

                            // Td style={{ paddingTop: 0 }} prop emulates vertical space when label was in cell instead of row
                            // and assumes adjacent empty cell has no paddingTop.
                            return (
                                <Tbody key={cve} isExpanded={isExpanded}>
                                    <Tr
                                        style={
                                            labels.length !== 0
                                                ? { borderBlockEnd: 'none' }
                                                : undefined
                                        }
                                    >
                                        <Td
                                            expand={{
                                                rowIndex,
                                                isExpanded,
                                                onToggle: () => expandedRowSet.toggle(cve),
                                            }}
                                        />
                                        <CVESelectionTd
                                            className={getVisibilityClass('cveSelection')}
                                            selectedCves={selectedCves}
                                            rowIndex={rowIndex}
                                            item={{
                                                cve,
                                                summary,
                                                numAffectedImages: affectedImageCount,
                                            }}
                                        />
                                        <Td dataLabel="CVE" modifier="nowrap">
                                            <Link
                                                to={urlBuilder.cveDetails(cve, vulnerabilityState)}
                                            >
                                                {cve}
                                            </Link>
                                        </Td>
                                        <Td
                                            dataLabel="Top CVSS"
                                            className={getVisibilityClass('topCvss')}
                                        >
                                            <CvssFormatted cvss={topCVSS} />
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('topNvdCvss')}
                                            dataLabel="Top NVD CVSS"
                                        >
                                            <CvssFormatted cvss={topNvdCVSS} />
                                        </Td>
                                        <Td
                                            className={getVisibilityClass('epssProbability')}
                                            dataLabel="EPSS probability"
                                        >
                                            {formatEpssProbabilityAsPercent(topEpssProbability)}
                                        </Td>
                                        <Td
                                            dataLabel="Affected images"
                                            className={getVisibilityClass('affectedImages')}
                                            modifier="nowrap"
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
                                        <Td
                                            className={getVisibilityClass('requestDetails')}
                                            dataLabel="Request details"
                                        >
                                            {vulnerabilityState !== 'OBSERVED' && (
                                                <ExceptionDetailsCell
                                                    cve={cve}
                                                    vulnerabilityState={vulnerabilityState}
                                                />
                                            )}
                                        </Td>
                                        <Td
                                            isActionCell
                                            className={getVisibilityClass('rowActions')}
                                        >
                                            <ActionsColumn
                                                popperProps={ACTION_COLUMN_POPPER_PROPS}
                                                items={createTableActions({
                                                    cve,
                                                    summary,
                                                    numAffectedImages: affectedImageCount,
                                                })}
                                            />
                                        </Td>
                                    </Tr>
                                    {labels.length !== 0 && (
                                        <Tr
                                            style={
                                                isExpanded ? { borderBlockEnd: 'none' } : undefined
                                            }
                                        >
                                            <Td colSpan={2} />
                                            <Td colSpan={colSpan - 2} style={{ paddingTop: 0 }}>
                                                <LabelGroup numLabels={labels.length}>
                                                    {labels}
                                                </LabelGroup>
                                            </Td>
                                        </Tr>
                                    )}
                                    <Tr isExpanded={isExpanded}>
                                        <Td />
                                        <Td colSpan={colSpan - 1}>
                                            <ExpandableRowContent>
                                                {isExpanded && (
                                                    <CVESummaryContent cve={cve} query={queryString} />
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
