import { useCallback, useRef } from 'react';
import { useLocation, useHistory } from 'react-router-dom';
import isEqual from 'lodash/isEqual';
import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

export type QueryValue = undefined | string | string[] | qs.ParsedQs | qs.ParsedQs[];

// Note that when we upgrade React Router and 'history' we can probably import a more accurate version of this type
export type Action = 'push' | 'replace';

export type UseURLParameterResult = [
    QueryValue,
    (newValue: QueryValue, historyAction?: Action) => void,
];

/**
 * Hook to handle reading and writing of a piece of state in the page's URL query parameters.
 *
 * The return value of this hook follows the `useState` convention, returning a 2-length
 * array where the first item is the state value of the URL parameter and the second
 * value is a setter function to change that value.
 *
 * Both the returned state and setter function maintain referential equality across
 * calls as long as the state in the URL does not change.
 *
 * @param keyPrefix The key value of the url parameter to manage
 * @param defaultValue A default value to use when the parameter is not available in the URL
 *
 * @returns [value, setterFn]
 */
function useURLParameter(keyPrefix: string, defaultValue: QueryValue): UseURLParameterResult {
    const history = useHistory();
    const location = useLocation();
    // We use an internal Ref here so that calling code that depends on the
    // value returned by this hook can detect updates. e.g. When used in the
    // dependency array of a `useEffect`.
    const internalValue = useRef(defaultValue);
    // memoize the setter function to retain referential equality as long
    // as the URL parameters do not change
    const setValue = useCallback(
        (newValue: QueryValue, historyAction: Action = 'push') => {
            // Note that we use the version of `location` on `history` here, since it is mutable (compared
            // to the immutable `location` when used directly.) In this case, we aren't looking
            // for a reference change to trigger a rerender, we are looking for the current up-to-date
            // location search parameters.
            //
            // https://v5.reactrouter.com/web/api/history/history-is-mutable
            const previousQuery = getQueryObject(history.location.search) || {};
            const newQuery = {
                ...previousQuery,
                [keyPrefix]: newValue,
            };

            // If the value passed in is `undefined`, don't display it in the URL at all
            if (typeof newValue === 'undefined') {
                delete newQuery[keyPrefix];
            }

            // Do not change history states if setter is called with current value
            if (!isEqual(previousQuery[keyPrefix], newValue)) {
                history[historyAction]({ search: getQueryString(newQuery) });
            }
        },
        [keyPrefix, history]
    );

    const nextValue = getQueryObject(location.search)[keyPrefix] || defaultValue;

    // If the search filter has changed, replace the object reference.
    if (!isEqual(internalValue.current, nextValue)) {
        internalValue.current = nextValue;
    }

    return [internalValue.current, setValue];
}

export default useURLParameter;
