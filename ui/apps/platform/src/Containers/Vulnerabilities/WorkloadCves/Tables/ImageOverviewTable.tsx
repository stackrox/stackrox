import React, { useState } from 'react';
import type { ReactNode } from 'react';
import { gql } from '@apollo/client';
import pluralize from 'pluralize';
import { ActionsColumn, IAction, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Flex, Label, LabelGroup } from '@patternfly/react-core';
import { EyeIcon } from '@patternfly/react-icons';
import isEmpty from 'lodash/isEmpty';

import { UseURLSortResult } from 'hooks/useURLSort';
import { DynamicColumnIcon } from 'Components/DynamicIcon';
import TooltipTh from 'Components/TooltipTh';
import DateDistance from 'Components/DateDistance';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import { TableUIState } from 'utils/getTableUIState';
import { ACTION_COLUMN_POPPER_PROPS } from 'constants/tables';
import {
    generateVisibilityForColumns,
    getHiddenColumnCount,
    ManagedColumns,
} from 'hooks/useManagedColumns';
import useIsScannerV4Enabled from 'hooks/useIsScannerV4Enabled';
import usePermissions from 'hooks/usePermissions';
import GenerateSbomModal, {
    getSbomGenerationStatusMessage,
} from '../../components/GenerateSbomModal';
import ImageNameLink from '../components/ImageNameLink';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { SignatureVerificationResult, VulnerabilitySeverityLabel, WatchStatus } from '../../types';
import ImageScanningIncompleteLabel from '../components/ImageScanningIncompleteLabel';
import VerifiedSignatureLabel, {
    getVerifiedSignatureInResults,
} from '../components/VerifiedSignatureLabel';
import getImageScanMessage from '../utils/getImageScanMessage';
import { getSeveritySortOptions } from '../../utils/sortUtils';

export const tableId = 'WorkloadCvesImageOverviewTable';

export const defaultColumns = {
    image: {
        title: 'Image',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
    cvesBySeverity: {
        title: 'CVEs by severity',
        isShownByDefault: true,
    },
    operatingSystem: {
        title: 'Operating system',
        isShownByDefault: true,
    },
    deployments: {
        title: 'Deployments',
        isShownByDefault: true,
    },
    age: {
        title: 'Age',
        isShownByDefault: true,
    },
    scanTime: {
        title: 'Scan time',
        isShownByDefault: true,
    },
    rowActions: {
        title: 'Row actions',
        isShownByDefault: true,
        isUntoggleAble: true,
    },
} as const;

export const imageListQuery = gql`
    query getImageList($query: String, $pagination: Pagination) {
        images(query: $query, pagination: $pagination) {
            id
            name {
                registry
                remote
                tag
                fullName
            }
            imageCVECountBySeverity(query: $query) {
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
            operatingSystem
            deploymentCount(query: $query)
            watchStatus
            metadata {
                v1 {
                    created
                }
            }
            scanTime
            scanNotes
            notes
            signatureVerificationData {
                results {
                    status
                    verifiedImageReferences
                    verifierId
                }
            }
        }
    }
`;

export type Image = {
    id: string;
    name: {
        registry: string;
        remote: string;
        tag: string;
        fullName: string;
    } | null;
    imageCVECountBySeverity: {
        critical: { total: number };
        important: { total: number };
        moderate: { total: number };
        low: { total: number };
        unknown: { total: number };
    };
    operatingSystem: string;
    deploymentCount: number;
    watchStatus: WatchStatus;
    metadata: {
        v1: {
            created: string | null;
        } | null;
    } | null;
    scanTime: string | null;
    scanNotes: string[];
    notes: string[];
    signatureVerificationData: {
        results: SignatureVerificationResult[];
    } | null;
};

export type ImageOverviewTableProps = {
    tableState: TableUIState<Image>;
    getSortParams: UseURLSortResult['getSortParams'];
    isFiltered: boolean;
    filteredSeverities?: VulnerabilitySeverityLabel[];
    hasWriteAccessForWatchedImage: boolean;
    onWatchImage: (imageName: string) => void;
    onUnwatchImage: (imageName: string) => void;
    onClearFilters: () => void;
    columnVisibilityState: ManagedColumns<keyof typeof defaultColumns>['columns'];
};

function ImageOverviewTable({
    tableState,
    getSortParams,
    isFiltered,
    filteredSeverities,
    hasWriteAccessForWatchedImage,
    onWatchImage,
    onUnwatchImage,
    onClearFilters,
    columnVisibilityState,
}: ImageOverviewTableProps) {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForImage = hasReadWriteAccess('Image'); // SBOM Generation mutates image scan state.
    const isScannerV4Enabled = useIsScannerV4Enabled();
    const getVisibilityClass = generateVisibilityForColumns(columnVisibilityState);
    const hiddenColumnCount = getHiddenColumnCount(columnVisibilityState);

    const colSpan = Object.values(defaultColumns).length - hiddenColumnCount;
    const [sbomTargetImage, setSbomTargetImage] = useState<string>();

    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th className={getVisibilityClass('image')} sort={getSortParams('Image')}>
                        Image
                    </Th>
                    <TooltipTh
                        className={getVisibilityClass('cvesBySeverity')}
                        tooltip="CVEs by severity across this image"
                        sort={getSortParams(
                            'CVEs By Severity',
                            getSeveritySortOptions(filteredSeverities)
                        )}
                    >
                        CVEs by severity
                        {isFiltered && <DynamicColumnIcon />}
                    </TooltipTh>
                    <Th
                        className={getVisibilityClass('operatingSystem')}
                        sort={getSortParams('Image OS')}
                    >
                        Operating system
                    </Th>
                    <Th className={getVisibilityClass('deployments')}>
                        Deployments
                        {isFiltered && <DynamicColumnIcon />}
                    </Th>
                    <Th
                        className={getVisibilityClass('age')}
                        sort={getSortParams('Image created time')}
                    >
                        Age
                    </Th>
                    <Th
                        className={getVisibilityClass('scanTime')}
                        sort={getSortParams('Image scan time')}
                    >
                        Scan time
                    </Th>
                    {/* eslint-disable-next-line generic/Th-defaultColumns */}
                    <Th className={getVisibilityClass('rowActions')}>
                        <span className="pf-v5-screen-reader">Row actions</span>
                    </Th>
                </Tr>
            </Thead>
            <TbodyUnified
                colSpan={colSpan}
                tableState={tableState}
                filteredEmptyProps={{ onClearFilters }}
                emptyProps={{ message: 'No images with observed CVEs were found in the system' }}
                renderer={({ data }) =>
                    data.map((image) => {
                        const {
                            id,
                            name,
                            imageCVECountBySeverity,
                            operatingSystem,
                            deploymentCount,
                            metadata,
                            watchStatus,
                            scanTime,
                            scanNotes,
                            notes,
                            signatureVerificationData,
                        } = image;
                        const criticalCount = imageCVECountBySeverity.critical.total;
                        const importantCount = imageCVECountBySeverity.important.total;
                        const moderateCount = imageCVECountBySeverity.moderate.total;
                        const lowCount = imageCVECountBySeverity.low.total;
                        const unknownCount = imageCVECountBySeverity.unknown.total;

                        const isWatchedImage = watchStatus === 'WATCHED';
                        const watchImageMenuText = isWatchedImage ? 'Unwatch image' : 'Watch image';
                        const watchImageMenuAction = isWatchedImage ? onUnwatchImage : onWatchImage;

                        const scanMessage = getImageScanMessage(notes, scanNotes);
                        const hasScanMessage = !isEmpty(scanMessage);

                        const rowActions: IAction[] = [];

                        if (hasWriteAccessForWatchedImage && name?.tag) {
                            rowActions.push({
                                title: watchImageMenuText,
                                onClick: () =>
                                    watchImageMenuAction(
                                        `${name.registry}/${name.remote}:${name.tag}`
                                    ),
                            });
                        }

                        if (hasWriteAccessForImage) {
                            const isAriaDisabled = !isScannerV4Enabled || hasScanMessage;
                            const description = getSbomGenerationStatusMessage({
                                isScannerV4Enabled,
                                hasScanMessage,
                            });

                            rowActions.push({
                                title: 'Generate SBOM',
                                isAriaDisabled,
                                description,
                                onClick: () => {
                                    setSbomTargetImage(name?.fullName);
                                },
                            });
                        }

                        const labels: ReactNode[] = [];
                        const verifiedSignatureResults = getVerifiedSignatureInResults(
                            signatureVerificationData?.results
                        );
                        if (verifiedSignatureResults.length !== 0) {
                            labels.push(
                                <VerifiedSignatureLabel
                                    key="verifiedSignatureResults"
                                    verifiedSignatureResults={verifiedSignatureResults}
                                    isCompact
                                    variant="outline"
                                />
                            );
                        }
                        if (isWatchedImage) {
                            labels.push(
                                <Label
                                    key="isWatchedImage"
                                    isCompact
                                    variant="outline"
                                    color="grey"
                                    icon={<EyeIcon />}
                                >
                                    Watched image
                                </Label>
                            );
                        }
                        if (hasScanMessage) {
                            labels.push(
                                <ImageScanningIncompleteLabel
                                    key="hasScanMessage"
                                    scanMessage={scanMessage}
                                />
                            );
                        }

                        // Td style={{ paddingTop: 0 }} prop emulates vertical space when label was in cell instead of row
                        // and assumes adjacent empty cell has no paddingTop.
                        return (
                            <Tbody
                                key={id}
                                style={{
                                    borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                                }}
                            >
                                <Tr>
                                    <Td className={getVisibilityClass('image')} dataLabel="Image">
                                        {name ? (
                                            <ImageNameLink name={name} id={id} />
                                        ) : (
                                            'Image name not available'
                                        )}
                                    </Td>
                                    <Td
                                        className={getVisibilityClass('cvesBySeverity')}
                                        dataLabel="CVEs by severity"
                                    >
                                        <SeverityCountLabels
                                            criticalCount={criticalCount}
                                            importantCount={importantCount}
                                            moderateCount={moderateCount}
                                            lowCount={lowCount}
                                            unknownCount={unknownCount}
                                            entity="image"
                                            filteredSeverities={filteredSeverities}
                                        />
                                    </Td>
                                    <Td
                                        dataLabel="Operating system"
                                        className={getVisibilityClass('operatingSystem')}
                                    >
                                        {operatingSystem || 'unknown'}
                                    </Td>
                                    <Td
                                        dataLabel="Deployments"
                                        className={getVisibilityClass('deployments')}
                                        modifier="nowrap"
                                    >
                                        {deploymentCount > 0 ? (
                                            <>
                                                {deploymentCount}{' '}
                                                {pluralize('deployment', deploymentCount)}
                                            </>
                                        ) : (
                                            <Flex>
                                                <div>0 deployments</div>
                                            </Flex>
                                        )}
                                    </Td>
                                    <Td dataLabel="Age" className={getVisibilityClass('age')}>
                                        {metadata?.v1?.created ? (
                                            <DateDistance
                                                date={metadata.v1.created}
                                                asPhrase={false}
                                            />
                                        ) : (
                                            'unknown'
                                        )}
                                    </Td>
                                    <Td
                                        dataLabel="Scan time"
                                        className={getVisibilityClass('scanTime')}
                                    >
                                        {scanTime ? <DateDistance date={scanTime} /> : 'unknown'}
                                    </Td>
                                    <Td className={getVisibilityClass('rowActions')} isActionCell>
                                        <ActionsColumn
                                            popperProps={ACTION_COLUMN_POPPER_PROPS}
                                            items={rowActions}
                                        />
                                    </Td>
                                </Tr>
                                {labels.length !== 0 && (
                                    <Tr>
                                        <Td colSpan={colSpan} style={{ paddingTop: 0 }}>
                                            <LabelGroup isCompact numLabels={labels.length}>
                                                {labels}
                                            </LabelGroup>
                                        </Td>
                                    </Tr>
                                )}
                            </Tbody>
                        );
                    })
                }
            />
            {sbomTargetImage && (
                <GenerateSbomModal
                    onClose={() => setSbomTargetImage(undefined)}
                    imageName={sbomTargetImage}
                />
            )}
        </Table>
    );
}

export default ImageOverviewTable;
