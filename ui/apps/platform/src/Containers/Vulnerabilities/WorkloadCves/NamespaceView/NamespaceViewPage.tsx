import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Bullseye,
    Button,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Pagination,
    Spinner,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { FileAltIcon, SearchIcon } from '@patternfly/react-icons';
import { gql, useQuery } from '@apollo/client';

import { vulnerabilitiesWorkloadCvesPath } from 'routePaths';
import { getTableUIState } from 'utils/getTableUIState';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import {
    CLUSTER_SEARCH_OPTION,
    NAMESPACE_LABEL_SEARCH_OPTION,
    NAMESPACE_SEARCH_OPTION,
    SearchOption,
} from 'Containers/Vulnerabilities/searchOptions';

import TableErrorComponent from 'Components/PatternFly/TableErrorComponent';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import FilterAutocompleteSelect from 'Containers/Vulnerabilities/components/FilterAutocomplete';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import KeyValueListModal from 'Components/KeyValueListModal';
import DeploymentFilterLink from './DeploymentFilterLink';

type Namespace = {
    metadata: {
        id: string;
        name: string;
        clusterId: string;
        clusterName: string;
        labels: {
            key: string;
            value: string;
        }[];
        priority: number;
    };
    deploymentCount: number;
};

const namespacesQuery = gql`
    query getNamespaceViewNamespaces($query: String, $pagination: Pagination) {
        namespaces(query: $query, pagination: $pagination) {
            metadata {
                id
                name
                clusterId
                clusterName
                labels {
                    key
                    value
                }
                priority
            }
            deploymentCount(query: $query)
        }
    }
`;

const defaultSearchFilters = {
    'Vulnerability State': 'OBSERVED',
};

const searchOptions: SearchOption[] = [
    NAMESPACE_SEARCH_OPTION,
    NAMESPACE_LABEL_SEARCH_OPTION,
    CLUSTER_SEARCH_OPTION,
];

const sortFields = ['Namespace Risk Priority', 'Namespace', 'Cluster', 'Deployment Count'];
const defaultSortOption = {
    field: sortFields[0],
    direction: 'asc',
} as const;

const pollInterval = 30000;

function NamespaceViewPage() {
    const { searchFilter, setSearchFilter } = useURLSearch();
    const { page, perPage, setPage, setPerPage } = useURLPagination(20);
    const { sortOption, getSortParams } = useURLSort({
        sortFields,
        defaultSortOption,
        onSort: () => setPage(1),
    });

    const {
        data,
        previousData,
        loading: isLoading,
        error,
    } = useQuery<{ namespaces: Namespace[] }>(namespacesQuery, {
        variables: {
            query: getRequestQueryStringForSearchFilter({
                ...searchFilter,
                ...defaultSearchFilters,
            }),
            pagination: {
                limit: perPage,
                offset: page - 1,
                sortOption,
            },
        },
        pollInterval,
    });

    const namespacesData = data?.namespaces ?? previousData?.namespaces;

    const tableUIState = getTableUIState({
        isLoading,
        data: namespacesData,
        error,
        searchFilter,
    });

    function onFilterChange() {
        setPage(1);
    }

    return (
        <>
            <PageTitle title="Workload CVEs - Namespace view" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={vulnerabilitiesWorkloadCvesPath}>
                        Workload CVEs
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Namespace view</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column' }}
                    alignItems={{ default: 'alignItemsFlexStart' }}
                >
                    <Title headingLevel="h1" className="pf-v5-u-mb-sm">
                        Namespace view
                    </Title>
                    <FlexItem>Discover and prioritize namespaces by risk priority</FlexItem>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <FilterAutocompleteSelect
                            searchFilter={searchFilter}
                            onFilterChange={(newFilter) => setSearchFilter(newFilter)}
                            searchOptions={searchOptions}
                        />
                        <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                            <Pagination
                                toggleTemplate={({ firstIndex, lastIndex }) => (
                                    <span>
                                        <b>
                                            {firstIndex} - {lastIndex}
                                        </b>{' '}
                                        of <b>many</b>
                                    </span>
                                )}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => setPerPage(newPerPage)}
                                isCompact
                            />
                        </ToolbarItem>
                        <ToolbarGroup aria-label="applied search filters" className="pf-u-w-100">
                            <SearchFilterChips
                                onFilterChange={onFilterChange}
                                filterChipGroupDescriptors={searchOptions.map(
                                    ({ label, value }) => {
                                        return {
                                            displayName: label,
                                            searchFilterName: value,
                                        };
                                    }
                                )}
                            />
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
                <TableComposable borders={false}>
                    <Thead noWrap>
                        <Tr>
                            <Th sort={getSortParams('Namespace')} width={30}>
                                Namespace
                            </Th>
                            <Th sort={getSortParams('Namespace Risk Priority')}>Risk priority</Th>
                            <Th sort={getSortParams('Cluster')}>Cluster</Th>
                            <Th sort={getSortParams('Deployment Count')}>Deployments</Th>
                            <Th>Labels</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {tableUIState.type === 'ERROR' && (
                            <Tr>
                                <Td colSpan={5}>
                                    <TableErrorComponent
                                        error={tableUIState.error}
                                        message="An error occurred. Try refreshing again"
                                    />
                                </Td>
                            </Tr>
                        )}
                        {tableUIState.type === 'LOADING' && (
                            <Tr>
                                <Td colSpan={5}>
                                    <Bullseye>
                                        <Spinner isSVG aria-label="Loading table data" />
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                        {tableUIState.type === 'EMPTY' && (
                            <Tr>
                                <Td colSpan={5}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="There are currently no namespaces"
                                            headingLevel="h2"
                                            icon={FileAltIcon}
                                        >
                                            <Text>There are currently no namespaces.</Text>
                                        </EmptyStateTemplate>
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                        {tableUIState.type === 'FILTERED_EMPTY' && (
                            <Tr>
                                <Td colSpan={5}>
                                    <Bullseye>
                                        <EmptyStateTemplate
                                            title="No results found"
                                            headingLevel="h2"
                                            icon={SearchIcon}
                                        >
                                            <Text>
                                                We couldn’t find any items matching your search
                                                criteria. Try adjusting your filters or search terms
                                                for better results
                                            </Text>
                                            <Button
                                                variant="link"
                                                onClick={() => {
                                                    setPage(1);
                                                    setSearchFilter({});
                                                }}
                                            >
                                                Clear search filters
                                            </Button>
                                        </EmptyStateTemplate>
                                    </Bullseye>
                                </Td>
                            </Tr>
                        )}
                        {(tableUIState.type === 'COMPLETE' || tableUIState.type === 'POLLING') &&
                            tableUIState.data.map((namespace) => {
                                const {
                                    metadata: { id, name, clusterName, labels, priority },
                                    deploymentCount,
                                } = namespace;

                                return (
                                    <Tr key={id}>
                                        <Td dataLabel="Namespace">{name}</Td>
                                        <Td dataLabel="Risk priority">{priority}</Td>
                                        <Td dataLabel="Cluster">{clusterName}</Td>
                                        <Td dataLabel="Deployments">
                                            <DeploymentFilterLink
                                                deploymentCount={deploymentCount}
                                                namespaceName={name}
                                                clusterName={clusterName}
                                            />
                                        </Td>
                                        <Td dataLabel="Labels">
                                            <KeyValueListModal type="label" keyValues={labels} />
                                        </Td>
                                    </Tr>
                                );
                            })}
                    </Tbody>
                </TableComposable>
            </PageSection>
        </>
    );
}

export default NamespaceViewPage;
