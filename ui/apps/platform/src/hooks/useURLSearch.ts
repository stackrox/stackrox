import { useRef } from 'react';
import isEqual from 'lodash/isEqual';

import { SearchFilter } from 'types/search';
import { isParsedQs } from 'utils/queryStringUtils';
import useURLParameter, { Action, QueryValue } from './useURLParameter';

type UseUrlSearchReturn = {
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter, historyAction?: Action) => void;
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

    return { searchFilter: searchFilterRef.current, setSearchFilter };
}

export default useURLSearch;
