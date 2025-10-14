import React, { useState, useMemo } from 'react';
import {
    Card,
    CardBody,
    PageSection,
    SearchInput,
    Select,
    SelectOption,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    EmptyState,
    EmptyStateHeader,
    EmptyStateIcon,
    Bullseye,
} from '@patternfly/react-core';
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
    const [searchValue, setSearchValue] = useState('');
    const [sortColumn, setSortColumn] = useState<SortColumn>('name');
    const [sortDirection, setSortDirection] = useState<SortDirection>('asc');
    const [clusterFilter, setClusterFilter] = useState<string[]>([]);
    const [namespaceFilter, setNamespaceFilter] = useState<string[]>([]);
    const [isClusterSelectOpen, setIsClusterSelectOpen] = useState(false);
    const [isNamespaceSelectOpen, setIsNamespaceSelectOpen] = useState(false);

    // Reset select states when baseImageId changes
    React.useEffect(() => {
        setIsClusterSelectOpen(false);
        setIsNamespaceSelectOpen(false);
        setClusterFilter([]);
        setNamespaceFilter([]);
    }, [baseImageId]);

    const deployments = MOCK_BASE_IMAGE_DEPLOYMENTS[baseImageId] || [];

    // Extract unique clusters and namespaces for filters
    const uniqueClusters = useMemo(() => {
        if (!deployments || deployments.length === 0) {
            return [];
        }
        return Array.from(new Set(deployments.map((d) => d.cluster))).sort();
    }, [deployments]);

    const uniqueNamespaces = useMemo(() => {
        if (!deployments || deployments.length === 0) {
            return [];
        }
        return Array.from(new Set(deployments.map((d) => d.namespace))).sort();
    }, [deployments]);

    const filteredDeployments = useMemo(() => {
        return deployments.filter((deployment) => {
            // Search filter
            const matchesSearch =
                !searchValue ||
                deployment.name.toLowerCase().includes(searchValue.toLowerCase()) ||
                deployment.cluster.toLowerCase().includes(searchValue.toLowerCase()) ||
                deployment.namespace.toLowerCase().includes(searchValue.toLowerCase()) ||
                deployment.image.toLowerCase().includes(searchValue.toLowerCase());

            // Cluster filter
            const matchesCluster =
                clusterFilter.length === 0 || clusterFilter.includes(deployment.cluster);

            // Namespace filter
            const matchesNamespace =
                namespaceFilter.length === 0 || namespaceFilter.includes(deployment.namespace);

            return matchesSearch && matchesCluster && matchesNamespace;
        });
    }, [deployments, searchValue, clusterFilter, namespaceFilter]);

    const sortedDeployments = useMemo(() => {
        return [...filteredDeployments].sort((a, b) => {
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
    }, [filteredDeployments, sortColumn, sortDirection]);

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

    const handleClusterSelect = (
        _event?: React.MouseEvent<Element, MouseEvent>,
        selection?: string | number
    ) => {
        if (selection) {
            const cluster = selection as string;
            setClusterFilter((prev) =>
                prev.includes(cluster) ? prev.filter((c) => c !== cluster) : [...prev, cluster]
            );
        }
    };

    const handleNamespaceSelect = (
        _event?: React.MouseEvent<Element, MouseEvent>,
        selection?: string | number
    ) => {
        if (selection) {
            const namespace = selection as string;
            setNamespaceFilter((prev) =>
                prev.includes(namespace)
                    ? prev.filter((n) => n !== namespace)
                    : [...prev, namespace]
            );
        }
    };

    const filteredSeverities: VulnerabilitySeverityLabel[] = [
        'Critical',
        'Important',
        'Moderate',
        'Low',
    ];

    if (deployments.length === 0) {
        return (
            <PageSection isFilled>
                <Bullseye>
                    <EmptyState>
                        <EmptyStateHeader
                            titleText="No deployments found"
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
                    <ToolbarGroup variant="filter-group">
                        <ToolbarItem variant="search-filter">
                            <SearchInput
                                placeholder="Search by deployment, cluster, namespace, or image"
                                value={searchValue}
                                onChange={(_event, value) => setSearchValue(value)}
                                onClear={() => setSearchValue('')}
                            />
                        </ToolbarItem>
                        <ToolbarItem>
                            <Select
                                key="cluster-select"
                                aria-label="Select clusters"
                                onToggle={(_event: React.MouseEvent | undefined, isOpen: boolean) =>
                                    setIsClusterSelectOpen(isOpen)
                                }
                                onSelect={handleClusterSelect}
                                selections={clusterFilter}
                                isOpen={isClusterSelectOpen}
                                placeholderText="Filter by cluster"
                                isDisabled={uniqueClusters.length <= 1}
                            >
                                {uniqueClusters.map((cluster) => (
                                    <SelectOption key={cluster} value={cluster}>
                                        {cluster}
                                    </SelectOption>
                                ))}
                            </Select>
                        </ToolbarItem>
                        <ToolbarItem>
                            <Select
                                key="namespace-select"
                                aria-label="Select namespaces"
                                onToggle={(_event: React.MouseEvent | undefined, isOpen: boolean) =>
                                    setIsNamespaceSelectOpen(isOpen)
                                }
                                onSelect={handleNamespaceSelect}
                                selections={namespaceFilter}
                                isOpen={isNamespaceSelectOpen}
                                placeholderText="Filter by namespace"
                                isDisabled={uniqueNamespaces.length <= 1}
                            >
                                {uniqueNamespaces.map((namespace) => (
                                    <SelectOption key={namespace} value={namespace}>
                                        {namespace}
                                    </SelectOption>
                                ))}
                            </Select>
                        </ToolbarItem>
                    </ToolbarGroup>
                </ToolbarContent>
            </Toolbar>

            <Card>
                <CardBody>
                    {sortedDeployments.length === 0 ? (
                        <Bullseye>
                            <EmptyState>
                                <EmptyStateHeader
                                    titleText="No deployments match the current filters"
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
                                            <Td dataLabel="Risk Priority">
                                                {deployment.riskPriority}
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

export default BaseImageDeploymentsTab;
