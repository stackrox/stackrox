import React, { useState } from 'react';
import { Table, Thead, Tr, Th, Tbody, Td, ActionsColumn, IAction } from '@patternfly/react-table';
import { Link } from 'react-router-dom';
import { Label, Modal, Button } from '@patternfly/react-core';
import { CheckCircleIcon, InProgressIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import DateDistance from 'Components/DateDistance';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { BaseImage, ScanningStatus } from '../types';
import { vulnerabilitiesBaseImagesPath } from 'routePaths';
import { VulnerabilitySeverityLabel } from '../../types';

type SortColumn = 'name' | 'status' | 'images' | 'deployments' | 'cves' | 'lastScanned';
type SortDirection = 'asc' | 'desc';

type BaseImageTableProps = {
    baseImages: BaseImage[];
    onRemove: (id: string) => void;
};

function getScanningStatusIcon(status: ScanningStatus) {
    switch (status) {
        case 'COMPLETED':
            return <CheckCircleIcon color="var(--pf-v5-global--success-color--100)" />;
        case 'IN_PROGRESS':
            return <InProgressIcon color="var(--pf-v5-global--info-color--100)" />;
        case 'FAILED':
            return <ExclamationCircleIcon color="var(--pf-v5-global--danger-color--100)" />;
        default:
            return null;
    }
}

function getScanningStatusLabel(status: ScanningStatus) {
    switch (status) {
        case 'COMPLETED':
            return (
                <Label color="green" icon={getScanningStatusIcon(status)} isCompact>
                    Completed
                </Label>
            );
        case 'IN_PROGRESS':
            return (
                <Label color="blue" icon={getScanningStatusIcon(status)} isCompact>
                    In Progress
                </Label>
            );
        case 'FAILED':
            return (
                <Label color="red" icon={getScanningStatusIcon(status)} isCompact>
                    Failed
                </Label>
            );
        default:
            return null;
    }
}

function BaseImageTable({ baseImages, onRemove }: BaseImageTableProps) {
    const [sortColumn, setSortColumn] = useState<SortColumn>('name');
    const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
    const [removeConfirmation, setRemoveConfirmation] = useState<{
        id: string;
        name: string;
    } | null>(null);

    const handleSort = (column: SortColumn) => {
        if (sortColumn === column) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortColumn(column);
            setSortDirection('asc');
        }
    };

    const getSortParams = (column: SortColumn) => ({
        sort: {
            sortBy: {
                index: 0,
                direction: sortDirection,
            },
            onSort: () => handleSort(column),
            columnIndex: 0,
        },
    });

    const sortedImages = [...baseImages].sort((a, b) => {
        let comparison = 0;

        switch (sortColumn) {
            case 'name':
                comparison = a.name.localeCompare(b.name);
                break;
            case 'status':
                comparison = a.scanningStatus.localeCompare(b.scanningStatus);
                break;
            case 'images':
                comparison = a.imageCount - b.imageCount;
                break;
            case 'deployments':
                comparison = a.deploymentCount - b.deploymentCount;
                break;
            case 'cves':
                comparison = a.cveCount.total - b.cveCount.total;
                break;
            case 'lastScanned':
                if (!a.lastScanned && !b.lastScanned) {
                    return 0;
                }
                if (!a.lastScanned) {
                    return 1;
                }
                if (!b.lastScanned) {
                    return -1;
                }
                comparison = new Date(a.lastScanned).getTime() - new Date(b.lastScanned).getTime();
                break;
            default:
                comparison = 0;
        }

        return sortDirection === 'asc' ? comparison : -comparison;
    });

    const handleRemoveClick = (id: string, name: string) => {
        setRemoveConfirmation({ id, name });
    };

    const confirmRemove = () => {
        if (removeConfirmation) {
            onRemove(removeConfirmation.id);
            setRemoveConfirmation(null);
        }
    };

    const filteredSeverities: VulnerabilitySeverityLabel[] = [
        'Critical',
        'Important',
        'Moderate',
        'Low',
    ];

    return (
        <>
            <Table variant="compact" borders>
                <Thead noWrap>
                    <Tr>
                        <Th sort={getSortParams('name')}>Base Image Name</Th>
                        <Th sort={getSortParams('status')}>Status</Th>
                        <Th sort={getSortParams('images')}>Images Using</Th>
                        <Th sort={getSortParams('deployments')}>Deployments</Th>
                        <Th sort={getSortParams('cves')}>CVEs</Th>
                        <Th sort={getSortParams('lastScanned')}>Last Scanned</Th>
                        <Th>
                            <span className="pf-v5-screen-reader">Row actions</span>
                        </Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {sortedImages.map((baseImage) => {
                        const rowActions: IAction[] = [
                            {
                                title: 'Remove',
                                onClick: () => handleRemoveClick(baseImage.id, baseImage.name),
                            },
                        ];

                        return (
                            <Tr key={baseImage.id}>
                                <Td dataLabel="Base Image Name">
                                    <Link to={`${vulnerabilitiesBaseImagesPath}/${baseImage.id}`}>
                                        {baseImage.name}
                                    </Link>
                                </Td>
                                <Td dataLabel="Status">
                                    {getScanningStatusLabel(baseImage.scanningStatus)}
                                </Td>
                                <Td dataLabel="Images Using">{baseImage.imageCount}</Td>
                                <Td dataLabel="Deployments">{baseImage.deploymentCount}</Td>
                                <Td dataLabel="CVEs">
                                    <SeverityCountLabels
                                        criticalCount={baseImage.cveCount.critical}
                                        importantCount={baseImage.cveCount.high}
                                        moderateCount={baseImage.cveCount.medium}
                                        lowCount={baseImage.cveCount.low}
                                        entity="image"
                                        filteredSeverities={filteredSeverities}
                                    />
                                </Td>
                                <Td dataLabel="Last Scanned">
                                    {baseImage.lastScanned ? (
                                        <DateDistance date={baseImage.lastScanned} />
                                    ) : (
                                        'Not scanned'
                                    )}
                                </Td>
                                <Td isActionCell>
                                    <ActionsColumn items={rowActions} />
                                </Td>
                            </Tr>
                        );
                    })}
                </Tbody>
            </Table>

            {removeConfirmation && (
                <Modal
                    variant="small"
                    title="Remove base image"
                    isOpen
                    onClose={() => setRemoveConfirmation(null)}
                    actions={[
                        <Button key="confirm" variant="danger" onClick={confirmRemove}>
                            Remove
                        </Button>,
                        <Button
                            key="cancel"
                            variant="link"
                            onClick={() => setRemoveConfirmation(null)}
                        >
                            Cancel
                        </Button>,
                    ]}
                >
                    Are you sure you want to stop tracking{' '}
                    <strong>{removeConfirmation.name}</strong>?
                </Modal>
            )}
        </>
    );
}

export default BaseImageTable;
