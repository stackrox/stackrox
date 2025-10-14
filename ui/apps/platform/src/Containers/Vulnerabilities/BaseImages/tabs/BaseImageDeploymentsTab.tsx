import React, { useState, useMemo } from 'react';
import { EmptyState, EmptyStateHeader, EmptyStateIcon, Bullseye } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { SearchIcon } from '@patternfly/react-icons';

import SeverityCountLabels from '../../components/SeverityCountLabels';
import { VulnerabilitySeverityLabel } from '../../types';
import { MOCK_BASE_IMAGE_DEPLOYMENTS } from '../mockData';

type BaseImageDeploymentsTabProps = {
    baseImageId: string;
};

type SortColumn = 'name' | 'namespace' | 'cluster' | 'image' | 'cves' | 'riskPriority';
type SortDirection = 'asc' | 'desc';

/**
 * Deployments tab for base image detail page - shows deployments using this base image
 */
function BaseImageDeploymentsTab({ baseImageId }: BaseImageDeploymentsTabProps) {
    const [sortColumn, setSortColumn] = useState<SortColumn>('name');
    const [sortDirection, setSortDirection] = useState<SortDirection>('asc');

    const deployments = useMemo(
        () => MOCK_BASE_IMAGE_DEPLOYMENTS[baseImageId] || [],
        [baseImageId]
    );

    const sortedDeployments = useMemo(() => {
        return [...deployments].sort((a, b) => {
            let comparison = 0;

            switch (sortColumn) {
                case 'name':
                    comparison = a.name.localeCompare(b.name);
                    break;
                case 'namespace':
                    comparison = a.namespace.localeCompare(b.namespace);
                    break;
                case 'cluster':
                    comparison = a.cluster.localeCompare(b.cluster);
                    break;
                case 'image':
                    comparison = a.image.localeCompare(b.image);
                    break;
                case 'cves':
                    comparison = a.cveCount.total - b.cveCount.total;
                    break;
                case 'riskPriority':
                    comparison = a.riskPriority - b.riskPriority;
                    break;
                default:
                    comparison = 0;
            }

            return sortDirection === 'asc' ? comparison : -comparison;
        });
    }, [deployments, sortColumn, sortDirection]);

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

    if (deployments.length === 0) {
        return (
            <Bullseye>
                <EmptyState>
                    <EmptyStateHeader
                        titleText="No deployments found"
                        icon={<EmptyStateIcon icon={SearchIcon} />}
                        headingLevel="h2"
                    />
                </EmptyState>
            </Bullseye>
        );
    }

    return (
        <>
            {sortedDeployments.length === 0 ? (
                <Bullseye>
                    <EmptyState>
                        <EmptyStateHeader
                            titleText="No deployments found"
                            icon={<EmptyStateIcon icon={SearchIcon} />}
                            headingLevel="h3"
                        />
                    </EmptyState>
                </Bullseye>
            ) : (
                <Table variant="compact" borders>
                    <Thead noWrap>
                        <Tr>
                            <Th sort={getSortParams('name')}>Deployment Name</Th>
                            <Th sort={getSortParams('namespace')}>Namespace</Th>
                            <Th sort={getSortParams('cluster')}>Cluster</Th>
                            <Th sort={getSortParams('image')}>Image</Th>
                            <Th sort={getSortParams('cves')}>CVEs</Th>
                            <Th sort={getSortParams('riskPriority')}>Risk Priority</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {sortedDeployments.map((deployment) => {
                            return (
                                <Tr key={deployment.deploymentId}>
                                    <Td dataLabel="Deployment Name">{deployment.name}</Td>
                                    <Td dataLabel="Namespace">{deployment.namespace}</Td>
                                    <Td dataLabel="Cluster">{deployment.cluster}</Td>
                                    <Td dataLabel="Image">{deployment.image}</Td>
                                    <Td dataLabel="CVEs">
                                        <SeverityCountLabels
                                            criticalCount={deployment.cveCount.critical}
                                            importantCount={deployment.cveCount.high}
                                            moderateCount={deployment.cveCount.medium}
                                            lowCount={deployment.cveCount.low}
                                            unknownCount={0}
                                            entity="deployment"
                                            filteredSeverities={filteredSeverities}
                                        />
                                    </Td>
                                    <Td dataLabel="Risk Priority">{deployment.riskPriority}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </Table>
            )}
        </>
    );
}

export default BaseImageDeploymentsTab;
