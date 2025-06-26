import { useRef } from 'react';
import isEqual from 'lodash/isEqual';

import { SearchFilter } from 'types/search';
import { isParsedQs } from 'utils/queryStringUtils';

import useURLParameter, { HistoryAction, QueryValue } from './useURLParameter';

export type SetSearchFilter = (newFilter: SearchFilter, historyAction?: HistoryAction) => void;

export type UseUrlSearchReturn = {
    searchFilter: SearchFilter;
    setSearchFilter: SetSearchFilter;
    buildSearchQuery: (filter: SearchFilter) => qs.ParsedQs;
};

function parseFilter(rawFilter: QueryValue): SearchFilter {
    const parsedFilter = {};

    if (!rawFilter || !isParsedQs(rawFilter)) {
        return parsedFilter;
    }

    Object.entries(rawFilter).forEach(([searchKey, searchVal]) => {
        if (typeof searchVal === 'string') {
            parsedFilter[searchKey] = searchVal;
        } else if (Array.isArray(searchVal)) {
            parsedFilter[searchKey] = [];
            searchVal.forEach((searchArrayEntry) => {
                if (typeof searchArrayEntry === 'string') {
                    parsedFilter[searchKey].push(searchArrayEntry);
                }
            });
        }
    });

    return parsedFilter;
}

function useURLSearch(keyPrefix = 's'): UseUrlSearchReturn {
    const [rawFilter, setSearchFilter] = useURLParameter(keyPrefix, {});
    const searchFilterRef = useRef<SearchFilter>({});
    const sanitizedFilter = parseFilter(rawFilter);

    if (!isEqual(searchFilterRef.current, sanitizedFilter)) {
        searchFilterRef.current = sanitizedFilter;
    }

    // TODO: explore alternatives. Might not be beneficial.
    // wanting a way to make a search query when using `Navigate()` that's
    // tightly coupled with useURLSearch/useURLParameter
    const buildSearchQuery = (filter: SearchFilter) => ({ [keyPrefix]: filter });

    return { searchFilter: searchFilterRef.current, setSearchFilter, buildSearchQuery };
}

export default useURLSearch;
