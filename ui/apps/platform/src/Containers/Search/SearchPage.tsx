import React, { useEffect, useState, ReactElement } from 'react';
import { useHistory, useLocation } from 'react-router-dom';
import {
    Alert,
    Bullseye,
    PageSection,
    Spinner,
    Stack,
    StackItem,
    Title,
} from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import SearchFilterInput from 'Components/SearchFilterInput';
import {
    SearchResponse,
    fetchGlobalSearchResults,
    getSearchOptionsForCategory,
} from 'services/SearchService';
import { SearchFilter } from 'types/search';
import { ORCHESTRATOR_COMPONENTS_KEY } from 'utils/orchestratorComponents';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { searchPath } from 'routePaths';

import SearchNavAndTable from './SearchNavAndTable';
import {
    parseQueryString,
    parseSearchFilter,
    stringifyQueryObject,
    stringifySearchFilter,
} from './searchQuery';

import './SearchPage.css';

function SearchPage(): ReactElement {
    const history = useHistory();
    const { search } = useLocation();

    /*
     * To avoid incomplete updates to the query string which produce uneeded items in browser history,
     * store a key which does not yet have a value in component state.
     */
    const [searchFilterKeyWithoutValue, setSearchFilterKeyWithoutValue] = useState<
        string | undefined
    >();

    const [isLoadingSearchOptions, setIsLoadingSearchOptions] = useState(false);
    const [searchOptions, setSearchOptions] = useState<string[]>([]);
    const [searchOptionsErrorMessage, setSeachOptionsErrorMessage] = useState<string | null>(null);

    const [isLoadingSearchResponse, setIsLoadingSearchResponse] = useState(false);
    const [searchResponse, setSearchResponse] = useState<SearchResponse | null>(null);
    const [searchResponseErrorMessage, setSeachResponseErrorMessage] = useState<string | null>(
        null
    );

    useEffect(() => {
        setIsLoadingSearchOptions(true);
        const { request, cancel } = getSearchOptionsForCategory();
        request
            .then(setSearchOptions)
            .catch((error) => {
                setSeachOptionsErrorMessage(getAxiosErrorMessage(error));
            })
            .finally(() => {
                setIsLoadingSearchOptions(false);
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
            const query = getRequestQueryStringForSearchFilter(
                localStorage.getItem(ORCHESTRATOR_COMPONENTS_KEY) !== 'true'
                    ? { ...parsedSearchFilter, 'Orchestrator Component': 'false' }
                    : parsedSearchFilter
            );

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

    function handleChangeSearchFilter(searchFilterNext: SearchFilter) {
        const searchFilterKeyWithoutValueNext = Object.keys(searchFilterNext).find(
            (key) => !searchFilterNext[key]
        );

        // If the changed search filter is complete, push the updated query string.
        if (!searchFilterKeyWithoutValueNext) {
            const queryString = stringifyQueryObject({
                searchFilter: searchFilterNext,
                navCategory,
            });
            const searchPathWithQueryString = `${searchPath}${queryString}`;

            // If the current search filter is empty, then replace, else push.
            if (stringifiedSearchFilter.length === 0) {
                history.replace(searchPathWithQueryString);
            } else {
                history.push(searchPathWithQueryString);
            }
        }

        setSearchFilterKeyWithoutValue(searchFilterKeyWithoutValueNext);
    }

    let content: ReactElement | null = null;

    if (isLoadingSearchOptions || isLoadingSearchResponse) {
        content = (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    } else if (searchOptions.length !== 0 && stringifiedSearchFilter.length === 0) {
        content = (
            <Alert variant="info" isInline title="Enter a new search filter">
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
            content = <Alert variant="info" isInline title="No results match the search filter" />;
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
            <Alert variant="danger" isInline title="Request failed for search results">
                {searchResponseErrorMessage}
            </Alert>
        );
    }

    const pageTitleItems = ['Search'];
    Object.keys(searchFilter).forEach((key) => {
        pageTitleItems.push(key);
    });

    return (
        <PageSection variant="light" id="search-page">
            <PageTitle title={pageTitleItems.join(' - ')} />
            <Stack hasGutter>
                <StackItem>
                    <Title headingLevel="h1" className="pf-u-mb-md">
                        Search
                    </Title>
                    {typeof searchOptionsErrorMessage === 'string' ? (
                        <Alert variant="danger" isInline title="Request failed for search options">
                            {searchOptionsErrorMessage}
                        </Alert>
                    ) : (
                        <SearchFilterInput
                            className="theme-light pf-search-shim z-xs-101"
                            handleChangeSearchFilter={handleChangeSearchFilter}
                            isDisabled={isLoadingSearchOptions || isLoadingSearchResponse}
                            placeholder="Filter resources"
                            searchFilter={
                                searchFilterKeyWithoutValue
                                    ? { ...searchFilter, [searchFilterKeyWithoutValue]: '' }
                                    : searchFilter
                            }
                            searchOptions={searchOptions}
                        />
                    )}
                </StackItem>
                <StackItem isFilled>{content}</StackItem>
            </Stack>
        </PageSection>
    );
}

export default SearchPage;
