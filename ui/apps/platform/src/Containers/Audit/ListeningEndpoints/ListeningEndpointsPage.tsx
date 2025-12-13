import { useCallback, useMemo, useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import {
    Bullseye,
    Button,
    Content,
    Divider,
    MenuToggle,
    PageSection,
    Pagination,
    Select,
    SelectList,
    SelectOption,
    Spinner,
    TextInputGroup,
    TextInputGroupMain,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
    debounce,
} from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { useQuery } from '@apollo/client';
import cloneDeep from 'lodash/cloneDeep';

import PageTitle from 'Components/PageTitle';
import EmptyStateTemplate from 'Components/EmptyStateTemplate/EmptyStateTemplate';
import SearchFilterChips from 'Components/CompoundSearchFilter/components/SearchFilterChips';
import SelectSingle from 'Components/SelectSingle/SelectSingle';
import { searchCategories } from 'constants/entityTypes';
import { toggleItemInArray } from 'utils/arrayUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import {
    applyRegexSearchModifiers,
    getRequestQueryStringForSearchFilter,
    searchValueAsArray,
} from 'utils/searchUtils';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import useURLSearch from 'hooks/useURLSearch';
import useRestQuery from 'hooks/useRestQuery';
import SEARCH_AUTOCOMPLETE_QUERY from 'queries/searchAutocomplete';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import type { SearchFilter } from 'types/search';
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

    const contextQueryString = getRequestQueryStringForSearchFilter(
        applyRegexSearchModifiers(filter)
    );
    const autocompleteSearchString = `${category}:${value ? `r/${value}` : ''}`;

    return contextQueryString
        ? `${contextQueryString}+${autocompleteSearchString}`
        : autocompleteSearchString;
}

const sortOptions = {
    sortFields: ['Deployment', 'Namespace', 'Cluster'],
    defaultSortOption: { field: 'Deployment', direction: 'asc' } as const,
};

function ListeningEndpointsPage() {
    const { page, perPage, setPage, setPerPage } = useURLPagination(10);
    const { sortOption, getSortParams } = useURLSort(sortOptions);
    const { searchFilter, setSearchFilter } = useURLSearch();
    const [entity, setEntity] = useState('Deployment');

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

    const [autocompleteOpen, setAutocompleteOpen] = useState(false);
    const [autocompleteInputValue, setAutocompleteInputValue] = useState('');
    const [debouncedSearchValue, setDebouncedSearchValue] = useState('');

    const [areAllRowsExpanded, setAllRowsExpanded] = useState(false);

    const variables = {
        query: getRequestQueryStringForAutocomplete(searchFilter, entity, debouncedSearchValue),
        categories: searchCategories[entity.toUpperCase()],
    };

    const { data: autoCompleteData } = useQuery(SEARCH_AUTOCOMPLETE_QUERY, { variables });

    function onEntitySelect(_id: string, selection: string) {
        setAutocompleteInputValue('');
        setDebouncedSearchValue('');
        setEntity(selection);
    }

    const updateSearchValue = useMemo(
        () => debounce((value: string) => setDebouncedSearchValue(value), 800),
        []
    );

    function onSearchFilterChange(searchFilter: SearchFilter) {
        setSearchFilter(searchFilter);
        setPage(1);
        setAutocompleteInputValue('');
        setDebouncedSearchValue('');
    }

    function onSelectAutocompleteValue(
        _event: ReactMouseEvent | undefined,
        value: string | number | undefined
    ) {
        if (typeof value === 'string') {
            const oldValue = searchValueAsArray(searchFilter[entity]);
            const newValue = toggleItemInArray(oldValue, value);
            setAutocompleteInputValue('');
            setDebouncedSearchValue('');
            setSearchFilter({ ...searchFilter, [entity]: newValue });
        }
    }

    const selectedValues = searchValueAsArray(searchFilter[entity]);
    const autocompleteOptions = autoCompleteData?.searchAutocomplete ?? [];

    // Show create option if user has typed something that doesn't exist
    const shouldShowCreateOption =
        autocompleteInputValue &&
        !autocompleteOptions.includes(autocompleteInputValue) &&
        !selectedValues.includes(autocompleteInputValue);

    const autocompleteToggle = (toggleRef: Ref<MenuToggleElement>) => (
        <MenuToggle
            ref={toggleRef}
            variant="typeahead"
            onClick={() => setAutocompleteOpen(!autocompleteOpen)}
            isExpanded={autocompleteOpen}
            isFullWidth
        >
            <TextInputGroup isPlain>
                <TextInputGroupMain
                    value={autocompleteInputValue}
                    onClick={() => setAutocompleteOpen(!autocompleteOpen)}
                    onChange={(_event, value) => {
                        setAutocompleteInputValue(value);
                        updateSearchValue(value);
                    }}
                    id="autocomplete-input"
                    placeholder={`Filter results by ${entity}`}
                    role="combobox"
                    isExpanded={autocompleteOpen}
                    aria-controls="autocomplete-listbox"
                />
            </TextInputGroup>
        </MenuToggle>
    );

    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection hasBodyWrapper={false} variant="default">
                <Title headingLevel="h1">Listening endpoints</Title>
                <Content component="p">
                    Audit listening endpoints of deployments in your clusters
                </Content>
            </PageSection>
            <Divider component="div" />
            <PageSection hasBodyWrapper={false}>
                <Toolbar className="pf-v6-u-pb-0">
                    <ToolbarContent>
                        <ToolbarGroup variant="filter-group" className="pf-v6-u-flex-grow-1">
                            <ToolbarItem>
                                <SelectSingle
                                    id="entity-filter"
                                    value={entity}
                                    handleSelect={onEntitySelect}
                                    toggleAriaLabel="Search entity selection menu toggle"
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
                                </SelectSingle>
                            </ToolbarItem>
                            <ToolbarItem className="pf-v6-u-flex-grow-1">
                                <Select
                                    id="autocomplete-filter"
                                    isOpen={autocompleteOpen}
                                    selected={selectedValues}
                                    onSelect={onSelectAutocompleteValue}
                                    onOpenChange={(isOpen) => setAutocompleteOpen(isOpen)}
                                    toggle={autocompleteToggle}
                                    className="pf-v6-u-flex-grow-1"
                                >
                                    <SelectList id="autocomplete-listbox">
                                        {autocompleteOptions.length > 0 ? (
                                            <>
                                                {autocompleteOptions.map((value) => (
                                                    <SelectOption key={value} value={value}>
                                                        {value}
                                                    </SelectOption>
                                                ))}
                                            </>
                                        ) : shouldShowCreateOption ? (
                                            <SelectOption value={autocompleteInputValue}>
                                                Add &quot;{autocompleteInputValue}&quot;
                                            </SelectOption>
                                        ) : (
                                            <SelectOption isDisabled>
                                                {autocompleteInputValue
                                                    ? 'No results found'
                                                    : 'Start typing to search'}
                                            </SelectOption>
                                        )}
                                    </SelectList>
                                </Select>
                            </ToolbarItem>
                        </ToolbarGroup>
                        <ToolbarItem variant="separator" />
                        <ToolbarGroup>
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
                        <ToolbarGroup className="pf-v6-u-w-100">
                            <SearchFilterChips
                                searchFilter={searchFilter}
                                onFilterChange={onSearchFilterChange}
                                filterChipGroupDescriptors={[
                                    { displayName: 'Deployment', searchFilterName: 'Deployment' },
                                    { displayName: 'Namespace', searchFilterName: 'Namespace' },
                                    { displayName: 'Cluster', searchFilterName: 'Cluster' },
                                ]}
                            />
                        </ToolbarGroup>
                    </ToolbarContent>
                </Toolbar>
            </PageSection>
            <PageSection
                hasBodyWrapper={false}
                id="listening-endpoints-page"
                isFilled
                padding={{ default: 'noPadding' }}
            >
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
                                            setPage(1);
                                            setAutocompleteInputValue('');
                                            setDebouncedSearchValue('');
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
            </PageSection>
        </>
    );
}

export default ListeningEndpointsPage;
