import React from 'react';
import { gql } from '@apollo/client';
import { ExpandableRowContent, Table, Tbody, Td, Thead, Th, Tr } from '@patternfly/react-table';

import useFeatureFlags from 'hooks/useFeatureFlags';
import useSet from 'hooks/useSet';
import { UseURLSortResult } from 'hooks/useURLSort';
import VulnerabilityFixableIconText from 'Components/PatternFly/IconText/VulnerabilityFixableIconText';
import VulnerabilitySeverityIconText from 'Components/PatternFly/IconText/VulnerabilitySeverityIconText';
import { VulnerabilityState } from 'types/cve.proto';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import CvssFormatted from 'Components/CvssFormatted';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TableUIState } from 'utils/getTableUIState';
import ExpandRowTh from 'Components/ExpandRowTh';
import { isNonEmptyArray } from 'utils/type.utils';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';
import {
    getIsSomeVulnerabilityFixable,
    getHighestCvssScore,
    getHighestNvdCvssScore,
    getHighestVulnerabilitySeverity,
    getEarliestDiscoveredAtTime,
} from '../../utils/vulnerabilityUtils';
import ImageNameLink from '../components/ImageNameLink';

import ImageComponentVulnerabilitiesTable, {
    ImageComponentVulnerability,
    convertToFlatImageComponentVulnerabilitiesFragment, // imageComponentVulnerabilitiesFragment
    imageMetadataContextFragment,
} from './ImageComponentVulnerabilitiesTable';
import { WatchStatus } from '../../types';
import PendingExceptionLabelLayout from '../components/PendingExceptionLabelLayout';

export const tableId = 'WorkloadCvesAffectedImagesTable';
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
    operatingSystem: {
        title: 'Operating system',
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
} as const;

export type ImageForCve = {
    id: string;
    name: {
        registry: string;
        remote: string;
        tag: string;
    } | null;
    metadata: {
        v1: {
            layers: {
                instruction: string;
                value: string;
            }[];
        } | null;
    } | null;
    operatingSystem: string;
    watchStatus: WatchStatus;
    imageComponents: (ImageComponentVulnerability & {
        imageVulnerabilities: (ImageComponentVulnerability['imageVulnerabilities'][number] & {
            discoveredAtImage: string;
            cvss: number;
            scoreVersion: string;
            nvdCvss: number;
            nvdScoreVersion: string; // for example, V3 or UNKNOWN_VERSION
        })[];
    })[];
};

// After release, replace temporary function
// with imagesForCveFragment
// that has unconditional imageComponentVulnerabilitiesFragment.
export function convertToFlatImagesForCveFragment(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
) {
    return gql`
        ${imageMetadataContextFragment}
        ${convertToFlatImageComponentVulnerabilitiesFragment(isFlattenCveDataEnabled)}
        fragment ImagesForCVE on Image {
            ...ImageMetadataContext

            operatingSystem
            watchStatus
            imageComponents(query: $query) {
                imageVulnerabilities(query: $query) {
                    discoveredAtImage
                    cvss
                    scoreVersion
                    nvdCvss
                    nvdScoreVersion
                }
                ...ImageComponentVulnerabilities
            }
        }
    `;
}

export type AffectedImagesTableProps = {
    tableState: TableUIState<ImageForCve>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    cve: string;
    vulnerabilityState: VulnerabilityState;
    onClearFilters: () => void;
    tableConfig: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function AffectedImagesTable({
    tableState,
    getSortParams,
    isFiltered,
    cve,
    vulnerabilityState,
    onClearFilters,
    tableConfig,
}: AffectedImagesTableProps) {
    const expandedRowSet = useSet<string>();

    const getVisibilityClass = generateVisibilityForColumns(tableConfig);
    const hiddenColumnCount = getHiddenColumnCount(tableConfig);

    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNvdCvssColumnEnabled = isFeatureFlagEnabled('ROX_SCANNER_V4');
    const colSpan = 8 + (isNvdCvssColumnEnabled ? 1 : 0) + -hiddenColumnCount;

    return (
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <ExpandRowTh />
                    <Th sort={getSortParams('Image')}>Image</Th>
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
                    <Th className={getVisibilityClass('cvss')}>CVSS</Th>
                    {isNvdCvssColumnEnabled && (
                        <Th className={getVisibilityClass('nvdCvss')}>NVD CVSS</Th>
                    )}
                    <Th
                        className={getVisibilityClass('operatingSystem')}
                        sort={getSortParams('Operating System')}
                    >
                        Operating system
                    </Th>
                    <Th className={getVisibilityClass('affectedComponents')}>
                        Affected components
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th className={getVisibilityClass('firstDiscovered')}>First discovered</Th>
                </Tr>
            </Thead>
            <TbodyUnified
                tableState={tableState}
                colSpan={colSpan}
                emptyProps={{ message: 'No images were found that are affected by this CVE' }}
                filteredEmptyProps={{ onClearFilters }}
                renderer={({ data }) =>
                    data.map((image, rowIndex) => {
                        const { id, name, operatingSystem, imageComponents } = image;
                        const vulnerabilities = imageComponents.flatMap(
                            (imageComponent) => imageComponent.imageVulnerabilities
                        );
                        const topSeverity = getHighestVulnerabilitySeverity(vulnerabilities);
                        const isFixableInImage = getIsSomeVulnerabilityFixable(vulnerabilities);
                        const { cvss, scoreVersion } = getHighestCvssScore(vulnerabilities);
                        const { nvdCvss, nvdScoreVersion } =
                            getHighestNvdCvssScore(vulnerabilities);
                        const hasPendingException = imageComponents.some((imageComponent) =>
                            imageComponent.imageVulnerabilities.some(
                                (imageVulnerability) => imageVulnerability.pendingExceptionCount > 0
                            )
                        );

                        const isExpanded = expandedRowSet.has(id);

                        return (
                            <Tbody key={id} isExpanded={isExpanded}>
                                <Tr>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded,
                                            onToggle: () => expandedRowSet.toggle(id),
                                        }}
                                    />
                                    <Td dataLabel="Image">
                                        {name ? (
                                            <PendingExceptionLabelLayout
                                                hasPendingException={hasPendingException}
                                                cve={cve}
                                                vulnerabilityState={vulnerabilityState}
                                            >
                                                <ImageNameLink name={name} id={id} />
                                            </PendingExceptionLabelLayout>
                                        ) : (
                                            'Image name not available'
                                        )}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveSeverity')}
                                        dataLabel="CVE severity"
                                        modifier="nowrap"
                                    >
                                        <VulnerabilitySeverityIconText severity={topSeverity} />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cveStatus')}
                                        dataLabel="CVE status"
                                        modifier="nowrap"
                                    >
                                        <VulnerabilityFixableIconText
                                            isFixable={isFixableInImage}
                                        />
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cvss')}
                                        dataLabel="CVSS"
                                        modifier="nowrap"
                                    >
                                        <CvssFormatted cvss={cvss} scoreVersion={scoreVersion} />
                                    </Td>

                                    {isNvdCvssColumnEnabled && (
                                        <Td
                                            className={getVisibilityClass('nvdCvss')}
                                            dataLabel="NVD CVSS"
                                            modifier="nowrap"
                                        >
                                            <CvssFormatted
                                                cvss={nvdCvss}
                                                scoreVersion={nvdScoreVersion}
                                            />
                                        </Td>
                                    )}
                                    <Td
                                        className={getVisibilityClass('operatingSystem')}
                                        dataLabel="Operating system"
                                    >
                                        {operatingSystem}
                                    </Td>
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
                                        {isNonEmptyArray(vulnerabilities) ? (
                                            <DateDistance
                                                date={getEarliestDiscoveredAtTime(vulnerabilities)}
                                            />
                                        ) : (
                                            'Not available'
                                        )}
                                    </Td>
                                </Tr>
                                <Tr isExpanded={isExpanded}>
                                    <Td />
                                    <Td colSpan={7}>
                                        <ExpandableRowContent>
                                            <ImageComponentVulnerabilitiesTable
                                                imageMetadataContext={image}
                                                componentVulnerabilities={image.imageComponents}
                                            />
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

export default AffectedImagesTable;
