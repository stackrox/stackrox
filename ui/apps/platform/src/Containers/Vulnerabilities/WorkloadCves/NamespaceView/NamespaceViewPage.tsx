import {
    Breadcrumb,
    BreadcrumbItem,
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

import { getTableUIState } from 'utils/getTableUIState';
import { getPaginationParams } from 'utils/searchUtils';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';

import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoumdSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import type { OnSearchCallback } from 'Components/CompoundSearchFilter/types';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import KeyValueListModal from 'Components/KeyValueListModal';
import TbodyUnified from 'Components/TableStateTemplates/TbodyUnified';
import useAnalytics, { WORKLOAD_CVE_FILTER_APPLIED } from 'hooks/useAnalytics';
import { createFilterTracker } from 'utils/analyticsEventTracking';
import type { SearchFilter } from 'types/search';
import { getRegexScopedQueryString, parseQuerySearchFilter } from '../../utils/searchUtils';
import { DEFAULT_VM_PAGE_SIZE } from '../../constants';
import useWorkloadCveViewContext from '../hooks/useWorkloadCveViewContext';
import { clusterSearchFilterConfig, namespaceSearchFilterConfig } from '../../searchFilterConfig';
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

const searchFilterConfig = [clusterSearchFilterConfig, namespaceSearchFilterConfig];

const sortFields = ['Namespace Risk Priority', 'Namespace', 'Cluster', 'Deployment Count'];
const defaultSortOption = {
    field: sortFields[0],
    direction: 'asc',
} as const;

const pollInterval = 30000;

function NamespaceViewPage() {
    const { analyticsTrack } = useAnalytics();
    const trackAppliedFilter = createFilterTracker(analyticsTrack);
    const { pageTitle, baseSearchFilter, urlBuilder } = useWorkloadCveViewContext();
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

    const onSearch: OnSearchCallback = (searchPayload) => {
        onFilterChange(updateSearchFilter(searchFilter, searchPayload));
        trackAppliedFilter(WORKLOAD_CVE_FILTER_APPLIED, searchPayload);
    };

    function onFilterChange(searchFilter: SearchFilter) {
        setSearchFilter(searchFilter);
        setPage(1);
    }

    return (
        <>
            <PageTitle title={`${pageTitle} - Namespace view`} />
            <PageSection type="breadcrumb">
                <Breadcrumb>
                    <BreadcrumbItemLink to={urlBuilder.vulnMgmtBase('')}>
                        {pageTitle}
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>Namespace view</BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <PageSection>
                <Flex
                    direction={{ default: 'column' }}
                    alignItems={{ default: 'alignItemsFlexStart' }}
                >
                    <Title headingLevel="h1" className="pf-v6-u-mb-sm">
                        Namespace view
                    </Title>
                    <FlexItem>Discover and prioritize namespaces by risk priority</FlexItem>
                </Flex>
            </PageSection>
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            defaultEntity="Namespace"
                            searchFilter={searchFilter}
                            onSearch={onSearch}
                        />
                        <ToolbarGroup aria-label="applied search filters" className="pf-v6-u-w-100">
                            <CompoumdSearchFilterLabels
                                attributesSeparateFromConfig={[]}
                                config={searchFilterConfig}
                                onFilterChange={onFilterChange}
                                searchFilter={searchFilter}
                            />
                        </ToolbarGroup>
                        <ToolbarGroup className="pf-v6-u-w-100">
                            <ToolbarItem variant="pagination" align={{ default: 'alignEnd' }}>
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
                                                    vulnMgmtBaseUrl={urlBuilder.vulnMgmtBase('')}
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
