import { useEffect, useState } from 'react';
import type { ReactElement } from 'react';
import { useLocation, useNavigate } from 'react-router-dom-v5-compat';
import {
    Alert,
    Bullseye,
    PageSection,
    Spinner,
    Stack,
    StackItem,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import CompoundSearchFilterLabels from 'Components/CompoundSearchFilter/components/CompoundSearchFilterLabels';
import {
    getSearchFilterConfigWithFeatureFlagDependency,
    updateSearchFilter,
} from 'Components/CompoundSearchFilter/utils/utils';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { fetchGlobalSearchResults, getSearchOptionsForCategory } from 'services/SearchService';
import type { SearchResponse } from 'services/SearchService';
import type { SearchFilter } from 'types/search';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { searchPath } from 'routePaths';

import SearchNavAndTable from './SearchNavAndTable';
import { searchFilterConfig } from './searchFilterConfig';
import {
    parseQueryString,
    parseSearchFilter,
    stringifyQueryObject,
    stringifySearchFilter,
} from './searchQuery';

import './SearchPage.css';

function SearchPage(): ReactElement {
    const navigate = useNavigate();
    const { search } = useLocation();
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const config = getSearchFilterConfigWithFeatureFlagDependency(
        isFeatureFlagEnabled,
        searchFilterConfig
    );

    /*
     * The curated compound search filter config is static, but the dynamic list of search
     * options is still requested so that a URL (for example, a saved bookmark) which uses a
     * search term outside the curated config continues to apply to the query.
     */
    const [searchOptions, setSearchOptions] = useState<string[]>([]);
    const [searchOptionsErrorMessage, setSeachOptionsErrorMessage] = useState<string | null>(null);

    const [isLoadingSearchResponse, setIsLoadingSearchResponse] = useState(false);
    const [searchResponse, setSearchResponse] = useState<SearchResponse | null>(null);
    const [searchResponseErrorMessage, setSeachResponseErrorMessage] = useState<string | null>(
        null
    );

    useEffect(() => {
        const { request, cancel } = getSearchOptionsForCategory();
        request.then(setSearchOptions).catch((error) => {
            setSeachOptionsErrorMessage(getAxiosErrorMessage(error));
        });

        return cancel;
    }, []);

    const { searchFilter, navCategory } = parseQueryString(search, searchOptions);
    const stringifiedSearchFilter = stringifySearchFilter(searchFilter);

    useEffect(() => {
        if (stringifiedSearchFilter.length === 0) {
            setSearchResponse(null);
        } else {
            setIsLoadingSearchResponse(true);
            setSeachResponseErrorMessage(null);

            const parsedSearchFilter = parseSearchFilter(stringifiedSearchFilter);
            const query = getRequestQueryStringForSearchFilter(parsedSearchFilter);

            fetchGlobalSearchResults({ query })
                .then(setSearchResponse)
                .catch((error) => {
                    setSearchResponse(null);
                    setSeachResponseErrorMessage(getAxiosErrorMessage(error));
                })
                .finally(() => {
                    setIsLoadingSearchResponse(false);
                });
        }
    }, [stringifiedSearchFilter]);

    // The compound search filter only emits complete search filters (an attribute always has a
    // value), so updating the URL is a simple push, or replace when there is no current filter.
    function handleChangeSearchFilter(searchFilterNext: SearchFilter) {
        const queryString = stringifyQueryObject({
            searchFilter: searchFilterNext,
            navCategory,
        });
        const searchPathWithQueryString = `${searchPath}${queryString}`;

        // If the current search filter is empty, then replace, else push.
        if (stringifiedSearchFilter.length === 0) {
            navigate(searchPathWithQueryString, { replace: true });
        } else {
            navigate(searchPathWithQueryString);
        }
    }

    let content: ReactElement | null = null;

    if (isLoadingSearchResponse) {
        content = (
            <Bullseye>
                <Spinner />
            </Bullseye>
        );
    } else if (stringifiedSearchFilter.length === 0) {
        content = (
            <Alert variant="info" isInline title="Enter a new search filter" component="p">
                <p>
                    Instead of a new search, you can go back in browser history to see previous
                    search results.
                </p>
                <p>
                    To preserve the scroll location in search results, you can right-click a link,
                    and then click Open Link in New Tab.
                </p>
            </Alert>
        );
    } else if (searchResponse) {
        if (searchResponse.results.length === 0) {
            content = (
                <Alert
                    variant="info"
                    isInline
                    title="No results match the search filter"
                    component="p"
                />
            );
        } else {
            content = (
                <SearchNavAndTable
                    activeNavCategory={navCategory}
                    searchFilter={searchFilter}
                    searchResponse={searchResponse}
                />
            );
        }
    } else if (typeof searchResponseErrorMessage === 'string') {
        content = (
            <Alert
                variant="danger"
                isInline
                title="Request failed for search results"
                component="p"
            >
                {searchResponseErrorMessage}
            </Alert>
        );
    }

    const pageTitleItems = ['Search'];
    Object.keys(searchFilter).forEach((key) => {
        pageTitleItems.push(key);
    });

    return (
        <PageSection hasBodyWrapper={false} id="search-page">
            <PageTitle title={pageTitleItems.join(' - ')} />
            <Stack hasGutter>
                <StackItem>
                    <Title headingLevel="h1" className="pf-v6-u-mb-md">
                        Search
                    </Title>
                    {typeof searchOptionsErrorMessage === 'string' && (
                        <Alert
                            variant="warning"
                            isInline
                            className="pf-v6-u-mb-md"
                            title="Request failed for the full list of search options"
                            component="p"
                        >
                            {searchOptionsErrorMessage} You can still filter using the options below.
                        </Alert>
                    )}
                    <Toolbar>
                        <ToolbarContent>
                            <CompoundSearchFilter
                                config={config}
                                defaultEntity="Deployment"
                                isDisabled={isLoadingSearchResponse}
                                searchFilter={searchFilter}
                                onSearch={(payload) =>
                                    handleChangeSearchFilter(
                                        updateSearchFilter(searchFilter, payload)
                                    )
                                }
                            />
                            <ToolbarGroup className="pf-v6-u-w-100">
                                <CompoundSearchFilterLabels
                                    attributesSeparateFromConfig={[]}
                                    config={config}
                                    onFilterChange={handleChangeSearchFilter}
                                    searchFilter={searchFilter}
                                />
                            </ToolbarGroup>
                        </ToolbarContent>
                    </Toolbar>
                </StackItem>
                <StackItem isFilled>{content}</StackItem>
            </Stack>
        </PageSection>
    );
}

export default SearchPage;
