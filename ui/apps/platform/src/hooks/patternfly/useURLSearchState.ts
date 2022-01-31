import { useLocation, useHistory } from 'react-router-dom';

import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

type SearchObject<T> = Record<string, T>;

type UseURLSearchStateResult<T> = [
    searchURLState: T | undefined,
    setSearchURLState: (newValue: T) => void
];

function useURLSearchState<T>(accessor: string): UseURLSearchStateResult<T> {
    const history = useHistory();
    const location = useLocation();
    const searchURLState: T | undefined = getQueryObject<SearchObject<T>>(location.search)?.[
        accessor
    ];

    function setSearchURLState(newValue) {
        const querySearchObject = getQueryObject<Record<string, string | string[]>>(
            location.search
        );
        const newSortOptionString = getQueryString({
            ...querySearchObject,
            [accessor]: newValue,
        });
        history.replace({
            search: newSortOptionString,
        });
    }

    return [searchURLState, setSearchURLState];
}

export default useURLSearchState;
