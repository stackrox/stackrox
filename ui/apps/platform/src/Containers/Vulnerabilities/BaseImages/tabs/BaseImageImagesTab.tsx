import React, { useState, useMemo } from 'react';
import {
    Card,
    CardBody,
    PageSection,
    SearchInput,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    EmptyState,
    EmptyStateHeader,
    EmptyStateIcon,
    Bullseye,
} from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';
import { Link } from 'react-router-dom';

import DateDistance from 'Components/DateDistance';
import SeverityCountLabels from '../../components/SeverityCountLabels';
import { VulnerabilitySeverityLabel } from '../../types';
import { vulnerabilitiesAllImagesPath } from 'routePaths';
import { MOCK_BASE_IMAGE_IMAGES } from '../mockData';

type BaseImageImagesTabProps = {
    baseImageId: string;
};

type SortColumn = 'name' | 'totalCves' | 'baseCves' | 'appCves' | 'deployments' | 'lastScanned';
type SortDirection = 'asc' | 'desc';

/**
 * Images tab for base image detail page - shows application images using this base
 */
function BaseImageImagesTab({ baseImageId }: BaseImageImagesTabProps) {
    const [searchValue, setSearchValue] = useState('');
    const [sortColumn, setSortColumn] = useState<SortColumn>('name');
    const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

    const images = MOCK_BASE_IMAGE_IMAGES[baseImageId] || [];

    const filteredImages = useMemo(() => {
        return images.filter(
            (image) =>
                !searchValue ||
                image.name.toLowerCase().includes(searchValue.toLowerCase()) ||
                image.sha.toLowerCase().includes(searchValue.toLowerCase())
        );
    }, [images, searchValue]);

    const sortedImages = useMemo(() => {
        return [...filteredImages].sort((a, b) => {
            let comparison = 0;

            switch (sortColumn) {
                case 'name':
                    comparison = a.name.localeCompare(b.name);
                    break;
                case 'totalCves':
                    comparison = a.cveCount.total - b.cveCount.total;
                    break;
                case 'baseCves':
                    comparison = a.cveCount.baseImageCves - b.cveCount.baseImageCves;
                    break;
                case 'appCves':
                    comparison = a.cveCount.applicationLayerCves - b.cveCount.applicationLayerCves;
                    break;
                case 'deployments':
                    comparison = a.deploymentCount - b.deploymentCount;
                    break;
                case 'lastScanned':
                    comparison =
                        new Date(a.lastScanned).getTime() - new Date(b.lastScanned).getTime();
                    break;
                default:
                    comparison = 0;
            }

            return sortDirection === 'asc' ? comparison : -comparison;
        });
    }, [filteredImages, sortColumn, sortDirection]);

    const handleSort = (column: SortColumn) => {
        if (sortColumn === column) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortColumn(column);
            setSortDirection('asc');
        }
    };

    const getSortParams = (column: SortColumn) => ({
        sortBy: {
            index: 0,
            direction: sortDirection,
        },
        onSort: () => handleSort(column),
        columnIndex: 0,
    });

    const filteredSeverities: VulnerabilitySeverityLabel[] = [
        'Critical',
        'Important',
        'Moderate',
        'Low',
    ];

    if (images.length === 0) {
        return (
            <PageSection isFilled>
                <Bullseye>
                    <EmptyState>
                        <EmptyStateHeader
                            titleText="No application images found"
                            icon={<EmptyStateIcon icon={SearchIcon} />}
                            headingLevel="h2"
                        />
                    </EmptyState>
                </Bullseye>
            </PageSection>
        );
    }

    return (
        <PageSection isFilled>
            <Toolbar>
                <ToolbarContent>
                    <ToolbarItem variant="search-filter">
                        <SearchInput
                            placeholder="Search by image name or SHA"
                            value={searchValue}
                            onChange={(_event, value) => setSearchValue(value)}
                            onClear={() => setSearchValue('')}
                        />
                    </ToolbarItem>
                </ToolbarContent>
            </Toolbar>

            <Card>
                <CardBody>
                    {sortedImages.length === 0 ? (
                        <Bullseye>
                            <EmptyState>
                                <EmptyStateHeader
                                    titleText="No images match the current search"
                                    icon={<EmptyStateIcon icon={SearchIcon} />}
                                    headingLevel="h3"
                                />
                            </EmptyState>
                        </Bullseye>
                    ) : (
                        <Table variant="compact" borders>
                            <Thead noWrap>
                                <Tr>
                                    <Th sort={getSortParams('name')}>Image Name</Th>
                                    <Th>SHA</Th>
                                    <Th sort={getSortParams('totalCves')}>Total CVEs</Th>
                                    <Th sort={getSortParams('baseCves')}>Base CVEs</Th>
                                    <Th sort={getSortParams('appCves')}>App CVEs</Th>
                                    <Th sort={getSortParams('deployments')}>Deployments</Th>
                                    <Th sort={getSortParams('lastScanned')}>Last Scanned</Th>
                                </Tr>
                            </Thead>
                            <Tbody>
                                {sortedImages.map((image) => {
                                    return (
                                        <Tr key={image.imageId}>
                                            <Td dataLabel="Image Name">
                                                <Link
                                                    to={`${vulnerabilitiesAllImagesPath}/images/${image.imageId}?vulnerabilityState=OBSERVED`}
                                                >
                                                    {image.name}
                                                </Link>
                                            </Td>
                                            <Td dataLabel="SHA">
                                                <div
                                                    style={{
                                                        maxWidth: '200px',
                                                        overflow: 'hidden',
                                                        textOverflow: 'ellipsis',
                                                        whiteSpace: 'nowrap',
                                                    }}
                                                    title={image.sha}
                                                >
                                                    {image.sha}
                                                </div>
                                            </Td>
                                            <Td dataLabel="Total CVEs">
                                                <SeverityCountLabels
                                                    criticalCount={image.cveCount.critical}
                                                    importantCount={image.cveCount.high}
                                                    moderateCount={image.cveCount.medium}
                                                    lowCount={image.cveCount.low}
                                                    unknownCount={0}
                                                    entity="image"
                                                    filteredSeverities={filteredSeverities}
                                                />
                                            </Td>
                                            <Td dataLabel="Base CVEs">
                                                {image.cveCount.baseImageCves}
                                            </Td>
                                            <Td dataLabel="App CVEs">
                                                {image.cveCount.applicationLayerCves}
                                            </Td>
                                            <Td dataLabel="Deployments">{image.deploymentCount}</Td>
                                            <Td dataLabel="Last Scanned">
                                                <DateDistance date={image.lastScanned} />
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        </Table>
                    )}
                </CardBody>
            </Card>
        </PageSection>
    );
}

export default BaseImageImagesTab;
