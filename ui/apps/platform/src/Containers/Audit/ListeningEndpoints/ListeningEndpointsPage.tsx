import React, { useState } from 'react';
import {
    Bullseye,
    Button,
    Divider,
    PageSection,
    Pagination,
    SearchInput,
    Spinner,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import { useDeploymentListeningEndpoints } from './hooks/useDeploymentListeningEndpoints';
import ListeningEndpointsTable from './ListeningEndpointsTable';

const sortOptions = {
    sortFields: ['Deployment', 'Namespace', 'Cluster'],
    defaultSortOption: { field: 'Deployment', direction: 'asc' } as const,
};

function ListeningEndpointsPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [searchValue, setSearchValue] = useState(() => {
        const filter = searchFilter.Deployment;
        return Array.isArray(filter) ? filter.join(',') : filter;
    });

    const { data, error, loading } = useDeploymentListeningEndpoints(
        searchFilter,
        sortOption,
        page,
        perPage
    );

    function onSearchInputChange(_event, value) {
        setSearchValue(value);
    }

    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection variant="light">
                <Title headingLevel="h1">Listening endpoints</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection isFilled className="pf-u-display-flex pf-u-flex-direction-column">
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem variant="search-filter" className="pf-u-flex-grow-1">
                            <SearchInput
                                aria-label="Search by deployment"
                                placeholder="Search by deployment"
                                value={searchValue}
                                onChange={onSearchInputChange}
                                onSearch={() => setSearchFilter({ Deployment: searchValue })}
                                onClear={() => {
                                    setSearchValue('');
                                    setSearchFilter({});
                                }}
                            />
                        </ToolbarItem>
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
                            />
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                <div className="pf-u-background-color-100">
                    {error && (
                        <Bullseye>
                            <EmptyStateTemplate
                                title="Error loading deployments with listening endpoints"
                                headingLevel="h2"
                                icon={ExclamationCircleIcon}
                                iconClassName="pf-u-danger-color-100"
                            >
                                {getAxiosErrorMessage(error.message)}
                            </EmptyStateTemplate>
                        </Bullseye>
                    )}
                    {loading && (
                        <Bullseye>
                            <Spinner aria-label="Loading listening endpoints for deployments" />
                        </Bullseye>
                    )}
                    {!error && !loading && data && (
                        <>
                            {data.length === 0 ? (
                                <Bullseye>
                                    <EmptyStateTemplate
                                        title="No deployments with listening endpoints found"
                                        headingLevel="h2"
                                    >
                                        <Text>Clear any search value and try again</Text>
                                        <Button
                                            variant="link"
                                            onClick={() => {
                                                setPage(1);
                                                setSearchValue('');
                                                setSearchFilter({});
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
