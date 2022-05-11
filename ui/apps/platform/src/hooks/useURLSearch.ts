import { useRef } from 'react';
import isEqual from 'lodash/isEqual';

import { SearchFilter } from 'types/search';
import { parseFilter } from 'utils/searchUtils';
import useURLParameter from './useURLParameter';

type UseUrlSearchReturn = {
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
};

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
