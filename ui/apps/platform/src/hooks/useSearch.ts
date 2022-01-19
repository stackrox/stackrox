import { useLocation, useHistory } from 'react-router-dom';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

import { SearchFilter } from 'types/search';

type SearchObject = {
    search?: Record<string, string>;
};

function useSearch() {
    const history = useHistory();
    const location = useLocation();
    const searchFilter: SearchFilter = getQueryObject<SearchObject>(location.search).search || {};

    function setSearchFilter(newSearchFilter) {
        const newSearchString = getQueryString({
            search: newSearchFilter,
        });
        history.replace({
            search: newSearchString,
        });
    }

    return { searchFilter, setSearchFilter };
}

export default useSearch;
