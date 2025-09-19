import React from 'react';
import type { ReactNode } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { LabelGroup } from '@patternfly/react-core';
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
import { gql } from '@apollo/client';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import { CveBaseInfo, VulnerabilityState, isVulnerabilitySeverity } from 'types/cve.proto';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import useMap from 'hooks/useMap';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import TooltipTh from 'Components/TooltipTh';
import DateDistance from 'Components/DateDistance';
import ExpandRowTh from 'Components/ExpandRowTh';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TableUIState } from 'utils/getTableUIState';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';
import {
    getIsSomeVulnerabilityFixable,
    hasKnownExploit,
    hasKnownRansomwareCampaignUse,
} from '../../utils/vulnerabilityUtils';
import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    ImageMetadataContext,
    imageComponentVulnerabilitiesFragment,
} from './ImageComponentVulnerabilitiesTable';

import { CveSelectionsProps } from '../../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import KnownExploitLabel from '../../components/KnownExploitLabel';
import KnownRansomwareCampaignLabel from "../../components/KnownRansomwareCampaignLabel";
import PendingExceptionLabel from '../../components/PendingExceptionLabel';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { infoForEpssProbability } from './infoForTh';
// totalAdvisories out of scope for MVP
// import { formatEpssProbabilityAsPercent, formatTotalAdvisories } from './table.utils';
import { formatEpssProbabilityAsPercent } from './table.utils';

export const tableId = 'WorkloadCvesImageVulnerabilitiesTable';
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
    cveSeverity: {
        title: 'CVE severity',
        isShownByDefault: true,
    },
    cveStatus: {
        title: 'CVE status',
        isShownByDefault: true,
    },
    cvss: {
        title: 'CVSS',
        isShownByDefault: true,
    },
    nvdCvss: {
        title: 'NVD CVSS',
        isShownByDefault: true,
    },
    epssProbability: {
        title: 'EPSS probability',
        isShownByDefault: true,
    },
    // totalAdvisories out of scope for MVP
    /*
    totalAdvisories: {
        title: 'Advisories',
        isShownByDefault: true,
    },
    */
    affectedComponents: {
        title: 'Affected components',
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

export const imageVulnerabilitiesFragment = gql`
    ${imageComponentVulnerabilitiesFragment}
    fragment ImageVulnerabilityFields on ImageVulnerability {
        severity
        cve
        summary
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
        discoveredAtImage
        publishedOn
        pendingExceptionCount: exceptionCount(requestStatus: $statusesForExceptionCount)
        imageComponents(query: $query) {
            ...ImageComponentVulnerabilities
        }
    }
`;

export type ImageVulnerability = {
    severity: string;
    cve: string;
    summary: string;
    cvss: number;
    scoreVersion: string;
    nvdCvss: number;
    nvdScoreVersion: string; // for example, V3 or UNKNOWN_VERSION
    cveBaseInfo: CveBaseInfo;
    discoveredAtImage: string | null;
    publishedOn: string | null;
    pendingExceptionCount: number;
    imageComponents: ImageComponentVulnerability[];
};

export type ImageVulnerabilitiesTableProps = {
    imageMetadata: ImageMetadataContext | undefined;
    tableState: TableUIState<ImageVulnerability>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    selectedCves: ReturnType<typeof useMap<string, CveSelectionsProps['cves'][number]>>;
    vulnerabilityState: VulnerabilityState;
    createTableActions?: (cve: {
        cve: string;
        summary: string;
        numAffectedImages: number;
    }) => IAction[];
    onClearFilters: () => void;
    tableConfig: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function ImageVulnerabilitiesTable({
    imageMetadata,
    tableState,
    getSortParams,
    isFiltered,
    selectedCves,
    vulnerabilityState,
    createTableActions = () => [],
    onClearFilters,
    tableConfig,
}: ImageVulnerabilitiesTableProps) {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const { urlBuilder } = useWorkloadCveViewContext();
    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);
    const expandedRowSet = useSet<string>();

    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh className={getVisibilityClass('rowExpansion')} />
                    <CVESelectionTh
                        className={getVisibilityClass('cveSelection')}
                        selectedCves={selectedCves}
                    />
                    <Th className={getVisibilityClass('cve')} sort={getSortParams('CVE')}>
                        CVE
                    </Th>
                    <Th
                        className={getVisibilityClass('cveSeverity')}
                        sort={getSortParams('Severity')}
                    >
                        CVE severity
                    </Th>
                    <Th className={getVisibilityClass('cveStatus')}>
                        CVE status
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th className={getVisibilityClass('cvss')} sort={getSortParams('CVSS')}>
                        CVSS
                    </Th>
                    <Th className={getVisibilityClass('nvdCvss')}>NVD CVSS</Th>
                    <Th
                        className={getVisibilityClass('epssProbability')}
                        info={infoForEpssProbability}
                        sort={getSortParams('EPSS Probability')}
                    >
                        EPSS probability
                    </Th>
                    {/* isAdvisoryColumnEnabled && (
                        <Th className={getVisibilityClass('totalAdvisories')}>Advisories</Th>
                    ) */}
                    <Th className={getVisibilityClass('affectedComponents')}>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th className={getVisibilityClass('firstDiscovered')} modifier="nowrap">
                        First discovered
                    </Th>
                    <Th className={getVisibilityClass('publishedOn')} modifier="nowrap">
                        Published
                    </Th>
                    <TooltipTh
                        className={getVisibilityClass('requestDetails')}
                        tooltip="View information about this exception request"
                    >
                        Request details
                    </TooltipTh>
                    {/* eslint-disable-next-line generic/Th-defaultColumns */}
                    <Th className={getVisibilityClass('rowActions')}>
                        <span className="pf-v5-screen-reader">Row actions</span>
                    </Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{ message: 'There were no CVEs detected for this image' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((vulnerability, rowIndex) => {
                        const {
                            cve,
                            severity,
                            summary,
                            cvss,
                            scoreVersion,
                            nvdCvss,
                            nvdScoreVersion,
                            cveBaseInfo,
                            imageComponents,
                            discoveredAtImage,
                            publishedOn,
                            pendingExceptionCount,
                        } = vulnerability;
                        const vulnerabilities = imageComponents.flatMap(
                            (imageComponent) => imageComponent.imageVulnerabilities
                        );
                        const isFixableInImage = getIsSomeVulnerabilityFixable(vulnerabilities);
                        const epssProbability = cveBaseInfo?.epss?.epssProbability;
                        // totalAdvisories out of scope for MVP
                        // const totalAdvisories = undefined;

                        const labels: ReactNode[] = [];
                        if (
                            isFeatureFlagEnabled('ROX_SCANNER_V4') &&
                            isFeatureFlagEnabled('ROX_CISA_KEV') &&
                            hasKnownExploit(cveBaseInfo?.exploit)
                        ) {
                            labels.push(<KnownExploitLabel key="exploit" isCompact />);
                            if (hasKnownRansomwareCampaignUse(cveBaseInfo?.exploit)) {
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

                        const isExpanded = expandedRowSet.has(cve);

                        // Table borders={false} prop above and Tbody style prop below
                        // to prevent unwanted border between main row and conditional labels row.
                        //
                        // Td style={{ paddingTop: 0 }} prop emulates vertical space when label was in cell instead of row
                        // and assumes adjacent empty cell has no paddingTop.
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
                                        className={getVisibilityClass('rowExpansion')}
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
                                        item={{ cve, summary, numAffectedImages: 1 }}
                                    />
                                    <Td
                                        className={getVisibilityClass('cve')}
                                        dataLabel="CVE"
                                        modifier="nowrap"
                                    >
                                        <Link to={urlBuilder.cveDetails(cve, vulnerabilityState)}>
                                            {cve}
                                        </Link>
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveSeverity')}
                                        modifier="nowrap"
                                        dataLabel="CVE severity"
                                    >
                                        {isVulnerabilitySeverity(severity) && (
                                            <VulnerabilitySeverityIconText severity={severity} />
                                        )}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveStatus')}
                                        modifier="nowrap"
                                        dataLabel="CVE status"
                                    >
                                        <VulnerabilityFixableIconText
                                            isFixable={isFixableInImage}
                                        />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cvss')}
                                        modifier="nowrap"
                                        dataLabel="CVSS"
                                    >
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('nvdCvss')}
                                        modifier="nowrap"
                                        dataLabel="NVD CVSS"
                                    >
                                        <CvssFormatted
                                            cvss={nvdCvss ?? 0}
                                            scoreVersion={nvdScoreVersion ?? 'UNKNOWN_VERSION'}
                                        />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('epssProbability')}
                                        modifier="nowrap"
                                        dataLabel="EPSS probability"
                                    >
                                        {formatEpssProbabilityAsPercent(epssProbability)}
                                    </Td>
                                    {/* isAdvisoryColumnEnabled && (
                                        <Td
                                            className={getVisibilityClass('totalAdvisories')}
                                            modifier="nowrap"
                                            dataLabel="Advisories"
                                        >
                                            {formatTotalAdvisories(totalAdvisories)}
                                        </Td>
                                    ) */}
                                    <Td
                                        className={getVisibilityClass('affectedComponents')}
                                        dataLabel="Affected components"
                                    >
                                        {imageComponents.length === 1
                                            ? imageComponents[0].name
                                            : `${imageComponents.length} components`}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('firstDiscovered')}
                                        dataLabel="First discovered"
                                    >
                                        <DateDistance date={discoveredAtImage} />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('publishedOn')}
                                        dataLabel="Published"
                                    >
                                        {publishedOn ? (
                                            <DateDistance date={publishedOn} />
                                        ) : (
                                            'Not available'
                                        )}
                                    </Td>
                                    <Td className={getVisibilityClass('requestDetails')}>
                                        {vulnerabilityState !== 'OBSERVED' && (
                                            <ExceptionDetailsCell
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            />
                                        )}
                                    </Td>
                                    <Td className={getVisibilityClass('rowActions')} isActionCell>
                                        <ActionsColumn
                                            // menuAppendTo={() => document.body}
                                            items={createTableActions({
                                                cve,
                                                summary,
                                                numAffectedImages: 1,
                                            })}
                                        />
                                    </Td>
                                </Tr>
                                {labels.length !== 0 && (
                                    <Tr>
                                        <Td colSpan={2} />
                                        <Td colSpan={colSpan - 2} style={{ paddingTop: 0 }}>
                                            <LabelGroup numLabels={labels.length}>
                                                {labels}
                                            </LabelGroup>
                                        </Td>
                                    </Tr>
                                )}
                                <Tr isExpanded={isExpanded}>
                                    <Td className={getVisibilityClass('rowExpansion')} />
                                    <Td colSpan={colSpan - 1}>
                                        <ExpandableRowContent>
                                            <>
                                                {summary && (
                                                    <p className="pf-v5-u-mb-md">{summary}</p>
                                                )}
                                                {imageMetadata ? (
                                                    <ImageComponentVulnerabilitiesTable
                                                        imageMetadataContext={imageMetadata}
                                                        componentVulnerabilities={imageComponents}
                                                    />
                                                ) : (
                                                    <PartialCVEDataAlert />
                                                )}
                                            </>
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })
                }
            />
        </Table>
    );
}

export default ImageVulnerabilitiesTable;
