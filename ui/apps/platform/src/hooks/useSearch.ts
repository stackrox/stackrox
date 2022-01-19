import { useEffect, useState } from 'react';
import { useLocation, useHistory } from 'react-router-dom';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

import { SearchFilter } from 'types/search';

type SearchObject = {
    search?: Record<string, string>;
};

function useSearch() {
    const history = useHistory();
    const location = useLocation();
    const searchObject: SearchFilter = getQueryObject<SearchObject>(location.search).search || {};
    const [searchFilter, setSearchFilter] = useState<SearchFilter>(searchObject);

    useEffect(() => {
        const newSearchString = getQueryString({
            search: searchFilter,
        });
        history.replace({
            search: newSearchString,
        });
    }, [history, searchFilter]);

    return { searchFilter, setSearchFilter };
}

export default useSearch;
