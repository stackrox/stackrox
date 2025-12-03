import { useCallback, useState } from 'react';
import {
    Bullseye,
    Button,
    Content,
    Divider,
    PageSection,
    Pagination,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import { updateSearchFilter } from 'Components/CompoundSearchFilter/utils/utils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import useRestQuery from 'hooks/useRestQuery';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';
import { useDeploymentListeningEndpoints } from './hooks/useDeploymentListeningEndpoints';
import ListeningEndpointsTable from './ListeningEndpointsTable';
import { searchFilterConfig } from './searchFilterConfig';

const sortOptions = {
    sortFields: ['Deployment', 'Namespace', 'Cluster'],
    defaultSortOption: { field: 'Deployment', direction: 'asc' } as const,
};

function ListeningEndpointsPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();

    const deploymentCountFetcher = useCallback(
        () => fetchDeploymentsCount(searchFilter),
        [searchFilter]
    );

    const countQuery = useRestQuery(deploymentCountFetcher);

    const { data, error, isLoading } = useDeploymentListeningEndpoints(
        searchFilter,
        sortOption,
        page,
        perPage
    );

    const [areAllRowsExpanded, setAllRowsExpanded] = useState(false);

    function onSearchFilterChange(searchFilter: SearchFilter) {
        setSearchFilter(searchFilter);
        setPage(1);
    }

    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection hasBodyWrapper={false}>
                <Title headingLevel="h1">Listening endpoints</Title>
                <Content component="p" className="pf-v6-u-pt-xs">
                    Audit listening endpoints of deployments in your clusters
                </Content>
            </PageSection>
            <Divider component="div" />
            <PageSection
                hasBodyWrapper={false}
                isFilled
                className="pf-v6-u-display-flex pf-v6-u-flex-direction-column"
            >
                <Toolbar>
                    <ToolbarContent>
                        <CompoundSearchFilter
                            config={searchFilterConfig}
                            defaultEntity="Deployment"
                            searchFilter={searchFilter}
                            onSearch={(payload) =>
                                onSearchFilterChange(updateSearchFilter(searchFilter, payload))
                            }
                        />
                        <ToolbarGroup className="pf-v6-u-w-100">
                            <CompoundSearchFilterLabels
                                attributesSeparateFromConfig={[]}
                                config={searchFilterConfig}
                                onFilterChange={onSearchFilterChange}
                                searchFilter={searchFilter}
                            />
                        </ToolbarGroup>
                        <ToolbarGroup className="pf-v6-u-w-100">
                            <ToolbarItem variant="pagination" align={{ default: 'alignEnd' }}>
                                <Pagination
                                    itemCount={countQuery.data ?? 0}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => {
                                        setPerPage(newPerPage);
                                    }}
                                />
                            </ToolbarItem>
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
                <div className="pf-v6-u-background-color-100">
                    {error && (
                        <Bullseye>
                            <EmptyStateTemplate
                                title="Error loading deployments with listening endpoints"
                                headingLevel="h2"
                                icon={ExclamationCircleIcon}
                                status="danger"
                            >
                                {getAxiosErrorMessage(error.message)}
                            </EmptyStateTemplate>
                        </Bullseye>
                    )}
                    {isLoading && (
                        <Bullseye>
                            <Spinner aria-label="Loading listening endpoints for deployments" />
                        </Bullseye>
                    )}
                    {!error && !isLoading && data && (
                        <>
                            {data.length === 0 ? (
                                <Bullseye>
                                    <EmptyStateTemplate
                                        title="No deployments with listening endpoints found"
                                        headingLevel="h2"
                                    >
                                        <Content component="p">
                                            Clear any search value and try again
                                        </Content>
                                        <Button
                                            variant="link"
                                            onClick={() => {
                                                onSearchFilterChange({});
                                            }}
                                        >
                                            Clear search
                                        </Button>
                                    </EmptyStateTemplate>
                                </Bullseye>
                            ) : (
                                <ListeningEndpointsTable
                                    deployments={data}
                                    getSortParams={getSortParams}
                                    areAllRowsExpanded={areAllRowsExpanded}
                                    setAllRowsExpanded={setAllRowsExpanded}
                                />
                            )}
                        </>
                    )}
                </div>
            </PageSection>
        </>
    );
}

export default ListeningEndpointsPage;
