import React from 'react';
import {
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Pagination,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql, useQuery } from '@apollo/client';
import uniq from 'lodash/uniq';

import { getTableUIState } from 'utils/getTableUIState';
import { getPaginationParams, searchValueAsArray } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import { OnSearchPayload } from 'Components/CompoundSearchFilter/types';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import KeyValueListModal from 'Components/KeyValueListModal';
import { makeFilterChipDescriptors } from 'Components/CompoundSearchFilter/utils/utils';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import {
    clusterSearchFilterConfig,
    namespaceSearchFilterConfig,
} from 'Containers/Vulnerabilities/searchFilterConfig';
import { SearchFilter } from 'types/search';
import { getRegexScopedQueryString, parseQuerySearchFilter } from '../../utils/searchUtils';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
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
        annotations: {
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
                annotations {
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

const searchFilterConfig = [namespaceSearchFilterConfig, clusterSearchFilterConfig];

const filterChipGroupDescriptors = makeFilterChipDescriptors(searchFilterConfig);

const sortFields = ['Namespace Risk Priority', 'Namespace', 'Cluster', 'Deployment Count'];
const defaultSortOption = {
    field: sortFields[0],
    direction: 'asc',
} as const;

const pollInterval = 30000;

function NamespaceViewPage() {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);
    const { pageTitle, baseSearchFilter, getAbsoluteUrl } = useWorkloadCveViewContext();
    const { searchFilter, setSearchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter({
        ...baseSearchFilter,
        ...defaultSearchFilters,
        ...searchFilter,
    });
    const { page, perPage, setPage, setPerPage } = useURLPagination(DEFAULT_VM_PAGE_SIZE);
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
            query: getRegexScopedQueryString(querySearchFilter),
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        pollInterval,
    });

    const namespacesData = data?.namespaces ?? previousData?.namespaces;

    const tableState = getTableUIState({
        isLoading,
        data: namespacesData,
        error,
        searchFilter,
    });

    function onSearch(searchPayload: OnSearchPayload) {
        const { category, value, action } = searchPayload;
        const selectedSearchFilter = searchValueAsArray(searchFilter[category]);

        const newFilter = {
            ...searchFilter,
            [category]:
                action === 'ADD'
                    ? uniq([...selectedSearchFilter, value])
                    : selectedSearchFilter.filter((oldValue) => value !== oldValue),
        };

        onFilterChange(newFilter);
        trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
    }

    function onFilterChange(searchFilter: SearchFilter) {
        setSearchFilter(searchFilter);
        setPage(1);
    }

    return (
        <>
            <PageTitle title={`${pageTitle} - Namespace view`} />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItemLink to={getAbsoluteUrl('')}>{pageTitle}</BreadcrumbItemLink>
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
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            searchFilter={searchFilter}
                            onSearch={onSearch}
                        />
                        <ToolbarItem variant="pagination" align={{ default: 'alignRight' }}>
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
                        <ToolbarGroup aria-label="applied search filters" className="pf-v5-u-w-100">
                            <SearchFilterChips
                                searchFilter={searchFilter}
                                onFilterChange={onFilterChange}
                                filterChipGroupDescriptors={filterChipGroupDescriptors}
                            />
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
                <Table borders={false}>
                    <Thead noWrap>
                        <Tr>
                            <Th sort={getSortParams('Namespace')} width={30}>
                                Namespace
                            </Th>
                            <Th sort={getSortParams('Namespace Risk Priority')}>Risk priority</Th>
                            <Th sort={getSortParams('Cluster')}>Cluster</Th>
                            <Th sort={getSortParams('Deployment Count')}>Deployments</Th>
                            <Th>Labels</Th>
                            <Th>Annotations</Th>
                        </Tr>
                    </Thead>
                    <TbodyUnified
                        tableState={tableState}
                        colSpan={6}
                        errorProps={{
                            title: 'There was an error loading namespaces',
                        }}
                        emptyProps={{
                            message:
                                'No results found. Please try adjusting your search criteria or navigate back to a previous page.',
                        }}
                        filteredEmptyProps={{
                            title: 'No namespaces found',
                            message: 'Clear all filters and try again',
                        }}
                        renderer={({ data }) => (
                            <Tbody>
                                {data.map((namespace) => {
                                    const {
                                        metadata: {
                                            id,
                                            name,
                                            clusterName,
                                            labels,
                                            annotations,
                                            priority,
                                        },
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
                                                    vulnMgmtBaseUrl={getAbsoluteUrl('')}
                                                />
                                            </Td>
                                            <Td dataLabel="Labels">
                                                <KeyValueListModal
                                                    type="label"
                                                    keyValues={labels}
                                                />
                                            </Td>
                                            <Td dataLabel="Annotations">
                                                <KeyValueListModal
                                                    type="annotation"
                                                    keyValues={annotations}
                                                />
                                            </Td>
                                        </Tr>
                                    );
                                })}
                            </Tbody>
                        )}
                    />
                </Table>
            </PageSection>
        </>
    );
}

export default NamespaceViewPage;
