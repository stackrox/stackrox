import React, { useCallback, useMemo, useState } from 'react';
import {
    Bullseye,
    Button,
    Divider,
    PageSection,
    Pagination,
    Select,
    SelectOption,
    Spinner,
    Text,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    debounce,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate/EmptyStateTemplate';
import SearchFilterChips from 'Components/PatternFly/SearchFilterChips';
import { searchCategories } from 'constants/entityTypes';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { searchValueAsArray } from 'utils/searchUtils';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import useRestQuery from 'hooks/useRestQuery';
import useSelectToggle from 'hooks/patternfly/useSelectToggle';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import { SearchFilter } from 'types/search';
import { useDeploymentListeningEndpoints } from './hooks/useDeploymentListeningEndpoints';
import ListeningEndpointsTable from './ListeningEndpointsTable';

import './ListeningEndpointsPage.css';

/**
 * Return request query string for autocomplete searches. Results will be scoped based
 * on applied filters that are of a higher scope than the autocomplete category.
 * e.g. If the user is filtering by Namespace, the autocomplete results will be scoped
 *      to the the selected Cluster, if it exists. Filters of the same scope or lower
 *      will not be included in the query string.
 * @param searchFilter The current search filter
 * @param category The category of the autocomplete search
 * @param value The value of the autocomplete search
 * @returns The request query string
 */
export function getRequestQueryStringForAutocomplete(
    searchFilter: SearchFilter,
    category: string,
    value: string
): string {
    const filter = cloneDeep(searchFilter);

    // Do not include lower scoped filters in the query string
    if (category === 'Cluster' || category === 'Namespace') {
        delete filter.Deployment;
    }
    if (category === 'Cluster') {
        delete filter.Namespace;
    }
    // Do not include any existing filters for the autocomplete category
    // e.g. If the user is filtering by Namespace, do not include the Namespace
    //      filter in the query string
    delete filter[category];

    return Object.entries(filter)
        .map(([key, val]) => `${key}:${Array.isArray(val) ? val.join(',') : val ?? ''}`)
        .concat(`${category}:${value}`)
        .join('+');
}

const sortOptions = {
    sortFields: ['Deployment', 'Namespace', 'Cluster'],
    defaultSortOption: { field: 'Deployment', direction: 'asc' } as const,
};

function ListeningEndpointsPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [searchValue, setSearchValue] = useState('');
    const [entity, setEntity] = useState('Deployment');

    const deploymentCountFetcher = useCallback(
        () => fetchDeploymentsCount(searchFilter),
        [searchFilter]
    );

    const countQuery = useRestQuery(deploymentCountFetcher);

    const { data, error, loading } = useDeploymentListeningEndpoints(
        searchFilter,
        sortOption,
        page,
        perPage
    );

    const entityToggle = useSelectToggle();
    const autocompleteToggle = useSelectToggle();

    const [areAllRowsExpanded, setAllRowsExpanded] = useState<boolean>(false);

    const variables = {
        query: getRequestQueryStringForAutocomplete(searchFilter, entity, searchValue),
        categories: searchCategories[entity.toUpperCase()],
    };

    const { data: autoCompleteData } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    function onEntitySelect(e, selection) {
        setSearchValue('');
        setEntity(selection);
    }

    const updateSearchValue = useMemo(
        () => debounce((value: string) => setSearchValue(value), 800),
        []
    );

    function onSelectAutocompleteValue(value) {
        const oldValue = searchValueAsArray(searchFilter[entity]);
        const newValue = oldValue.includes(value)
            ? oldValue.filter((f) => f !== value)
            : [...oldValue, value];
        setSearchValue('');
        setSearchFilter({ ...searchFilter, [entity]: newValue });
    }

    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection variant="light">
                <Title headingLevel="h1">Listening endpoints</Title>
                <Text>Audit listening endpoints of deployments in your clusters</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                id="listening-endpoints-page"
                isFilled
                className="pf-u-display-flex pf-u-flex-direction-column"
            >
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarGroup className="pf-u-flex-grow-1">
                            <ToolbarItem
                                variant="search-filter"
                                className="pf-u-display-flex pf-u-flex-grow-1"
                            >
                                <Select
                                    variant="single"
                                    toggleAriaLabel="Search entity selection menu toggle"
                                    aria-label="Select an entity to filter by"
                                    onToggle={entityToggle.onToggle}
                                    onSelect={onEntitySelect}
                                    selections={entity}
                                    isOpen={entityToggle.isOpen}
                                    className="pf-u-flex-basis-0"
                                >
                                    <SelectOption key="Deployment" value="Deployment">
                                        Deployment
                                    </SelectOption>
                                    <SelectOption key="Namespace" value="Namespace">
                                        Namespace
                                    </SelectOption>
                                    <SelectOption key="Cluster" value="Cluster">
                                        Cluster
                                    </SelectOption>
                                </Select>
                                <Select
                                    typeAheadAriaLabel={`Search by ${entity}`}
                                    aria-label={`Filter by ${entity}`}
                                    onSelect={(e, value) => {
                                        onSelectAutocompleteValue(value);
                                    }}
                                    onToggle={autocompleteToggle.onToggle}
                                    isOpen={autocompleteToggle.isOpen}
                                    placeholderText={`Filter results by ${entity}`}
                                    variant="typeaheadmulti"
                                    isCreatable
                                    createText="Add"
                                    selections={searchFilter[entity]}
                                    onTypeaheadInputChanged={(val: string) => {
                                        updateSearchValue(val);
                                    }}
                                    className="pf-u-flex-grow-1"
                                >
                                    {autoCompleteData?.searchAutocomplete?.map((value) => (
                                        <SelectOption key={value} value={value} />
                                    ))}
                                </Select>
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarGroup>
                            <ToolbarItem variant="pagination" alignment={{ default: 'alignRight' }}>
                                <Pagination
                                    itemCount={countQuery.data ?? 0}
                                    page={page}
                                    perPage={perPage}
                                    onSetPage={(_, newPage) => setPage(newPage)}
                                    onPerPageSelect={(_, newPerPage) => {
                                        if ((countQuery.data ?? 0) < (page - 1) * newPerPage) {
                                            setPage(1);
                                        }
                                        setPerPage(newPerPage);
                                    }}
                                />
                            </ToolbarItem>
                        </ToolbarGroup>

                        <ToolbarGroup className="pf-u-w-100">
                            <SearchFilterChips
                                filterChipGroupDescriptors={[
                                    { displayName: 'Deployment', searchFilterName: 'Deployment' },
                                    { displayName: 'Namespace', searchFilterName: 'Namespace' },
                                    { displayName: 'Cluster', searchFilterName: 'Cluster' },
                                ]}
                            />
                        </ToolbarGroup>
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
