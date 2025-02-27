import React from 'react';
import { Link } from 'react-router-dom';
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
import { getIsSomeVulnerabilityFixable } from '../../utils/vulnerabilityUtils';
import { getWorkloadEntityPagePath } from '../../utils/searchUtils';
import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    ImageMetadataContext,
    imageComponentVulnerabilitiesFragment,
} from './ImageComponentVulnerabilitiesTable';

import { CveSelectionsProps } from '../../components/ExceptionRequestModal/CveSelections';
import CVESelectionTh from '../../components/CVESelectionTh';
import CVESelectionTd from '../../components/CVESelectionTd';
import ExceptionDetailsCell from '../components/ExceptionDetailsCell';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';
import PartialCVEDataAlert from '../../components/PartialCVEDataAlert';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { infoForEpssProbability } from './infoForTh';
import { formatEpssProbabilityAsPercent, formatTotalAdvisories } from './table.utils';

export const tableId = 'WorkloadCvesImageVulnerabilitiesTable';
export const defaultColumns = {
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
    totalAdvisories: {
        title: 'Advisories',
        isShownByDefault: true,
    },
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
    canSelectRows: boolean;
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
    canSelectRows,
    selectedCves,
    vulnerabilityState,
    createTableActions,
    onClearFilters,
    tableConfig,
}: ImageVulnerabilitiesTableProps) {
    const { getAbsoluteUrl } = useWorkloadCveViewContext();
    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);
    const expandedRowSet = useSet<string>();
    const showExceptionDetailsLink = vulnerabilityState !== 'OBSERVED';

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    // Omit for 4.7 release until CVE/advisory separatipn is available in 4.8 release.
    // const isEpssProbabilityColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const isEpssProbabilityColumnEnabled = false;
    const isAdvisoryColumnEnabled =
        isFeatureFlagEnabled('ROX_SCANNER_V4') &&
        isFeatureFlagEnabled('ROX_CVE_ADVISORY_SEPARATION');

    const colSpan =
        7 +
        (isNvdCvssColumnEnabled ? 1 : 0) +
        (isEpssProbabilityColumnEnabled ? 1 : 0) +
        (isAdvisoryColumnEnabled ? 1 : 0) +
        (canSelectRows ? 1 : 0) +
        (createTableActions ? 1 : 0) +
        (showExceptionDetailsLink ? 1 : 0) +
        -hiddenColumnCount;

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    {canSelectRows && <CVESelectionTh selectedCves={selectedCves} />}
                    <Th sort={getSortParams('CVE')}>CVE</Th>
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
                    {isNvdCvssColumnEnabled && (
                        <Th className={getVisibilityClass('nvdCvss')}>NVD CVSS</Th>
                    )}
                    {isEpssProbabilityColumnEnabled && (
                        <Th
                            className={getVisibilityClass('epssProbability')}
                            info={infoForEpssProbability}
                            sort={getSortParams('EPSS Probability')}
                        >
                            EPSS probability
                        </Th>
                    )}
                    {isAdvisoryColumnEnabled && (
                        <Th className={getVisibilityClass('totalAdvisories')}>Advisories</Th>
                    )}
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
                        const totalAdvisories = undefined;
                        const isExpanded = expandedRowSet.has(cve);

                        return (
                            <Tbody key={cve} isExpanded={isExpanded}>
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
                                            item={{ cve, summary, numAffectedImages: 1 }}
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
                                    {isNvdCvssColumnEnabled && (
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
                                    )}
                                    {isEpssProbabilityColumnEnabled && (
                                        <Td
                                            className={getVisibilityClass('epssProbability')}
                                            modifier="nowrap"
                                            dataLabel="EPSS probability"
                                        >
                                            {formatEpssProbabilityAsPercent(epssProbability)}
                                        </Td>
                                    )}
                                    {isAdvisoryColumnEnabled && (
                                        <Td
                                            className={getVisibilityClass('totalAdvisories')}
                                            modifier="nowrap"
                                            dataLabel="Advisories"
                                        >
                                            {formatTotalAdvisories(totalAdvisories)}
                                        </Td>
                                    )}
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
                                    {showExceptionDetailsLink && (
                                        <ExceptionDetailsCell
                                            cve={cve}
                                            vulnerabilityState={vulnerabilityState}
                                        />
                                    )}
                                    {createTableActions && (
                                        <Td isActionCell>
                                            <ActionsColumn
                                                // menuAppendTo={() => document.body}
                                                items={createTableActions({
                                                    cve,
                                                    summary,
                                                    numAffectedImages: 1,
                                                })}
                                            />
                                        </Td>
                                    )}
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={colSpan}>
                                        <ExpandableRowContent>
                                            {summary && imageMetadata ? (
                                                <>
                                                    <p className="pf-v5-u-mb-md">{summary}</p>
                                                    <ImageComponentVulnerabilitiesTable
                                                        imageMetadataContext={imageMetadata}
                                                        componentVulnerabilities={imageComponents}
                                                    />
                                                </>
                                            ) : (
                                                <PartialCVEDataAlert />
                                            )}
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
