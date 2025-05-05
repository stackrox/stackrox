import { createContext, useCallback, useContext, useRef } from 'react';
import { NavigateFunction, useLocation, useNavigate } from 'react-router-dom';
import isEqual from 'lodash/isEqual';

import { getQueryObject, getQueryString } from 'utils/queryStringUtils';

export type QueryValue = undefined | string | string[] | qs.ParsedQs | qs.ParsedQs[];

// Note that when we upgrade React Router and 'history' we can probably import a more accurate version of this type
export type HistoryAction = 'push' | 'replace';

export type UrlParameterUpdate = {
    keyPrefix: string;
    newValue: QueryValue;
    historyAction: HistoryAction;
};

/**
 * Given an array of URL parameter updates, apply them as a single operation to the URL.
 * If any of the updates in the batch specify a 'push' history action, the overall
 * action will be 'push', otherwise 'replace'.
 *
 * @param updates Url parameter updates that need to be applied to the URL
 * @param history The history object to use to apply the updates
 */
export function applyUpdatesToUrl(
    updates: UrlParameterUpdate[],
    search: string,
    pathname: string,
    navigate: NavigateFunction
) {
    const action = updates.some(({ historyAction }) => historyAction === 'push')
        ? 'push'
        : 'replace';

    const previousQuery = getQueryObject(search) || {};
    const newQuery = { ...previousQuery };

    updates.forEach(({ keyPrefix, newValue }) => {
        newQuery[keyPrefix] = newValue;

        // If the value passed in is `undefined`, don't display it in the URL at all
        if (typeof newValue === 'undefined') {
            delete newQuery[keyPrefix];
        }
    });

    // Do not change history states if setter is called with current value
    if (!isEqual(previousQuery, newQuery)) {
        if (action === 'push') {
            navigate(`${pathname}${getQueryString(newQuery)}`);
        } else if (action === 'replace') {
            navigate(`${pathname}${getQueryString(newQuery)}`, { replace: true });
        }
    }
}

/**
 * The default context object for scheduling URL parameter updates. This context
 * object schedules updates to be applied in a microtask, ensuring that multiple
 * updates to the same URL parameter are batched together.
 *
 * @returns A context object that can be used to schedule URL parameter updates to be applied
 */
function makeMicrotaskSchedulingContext() {
    let updates: UrlParameterUpdate[] = [];
    let isUpdateScheduled = false;

    function scheduleAndFlushUpdates(search: string, pathname: string, navigate: NavigateFunction) {
        queueMicrotask(() => {
            applyUpdatesToUrl(updates, search, pathname, navigate);
            updates = [];
            isUpdateScheduled = false;
        });
    }

    return {
        addUrlParameterUpdate: (
            update: UrlParameterUpdate,
            search: string,
            pathname: string,
            navigate: NavigateFunction
        ) => {
            updates = [...updates, update];
            if (!isUpdateScheduled) {
                scheduleAndFlushUpdates(search, pathname, navigate);
            }
            isUpdateScheduled = true;
        },
    };
}

export const UrlParameterUpdateContext = createContext(makeMicrotaskSchedulingContext());

export type UseURLParameterResult = [
    QueryValue,
    (newValue: QueryValue, historyAction?: HistoryAction) => void,
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
    const { addUrlParameterUpdate } = useContext(UrlParameterUpdateContext);
    const location = useLocation();
    const navigate = useNavigate();

    // Using `useRef` for `location` to prevent re-creation of the callback on every render,
    // ensuring the latest `location` value is used without causing unnecessary re-renders.
    const locationRef = useRef(location);
    locationRef.current = location;

    // We use an internal Ref here so that calling code that depends on the
    // value returned by this hook can detect updates. e.g. When used in the
    // dependency array of a `useEffect`.
    const internalValue = useRef(defaultValue);
    // memoize the setter function to retain referential equality as long
    // as the URL parameters do not change

    const setValue = useCallback(
        (newValue: QueryValue, historyAction: HistoryAction = 'push') => {
            const { search, pathname } = locationRef.current;
            addUrlParameterUpdate(
                { historyAction, keyPrefix, newValue },
                search,
                pathname,
                navigate
            );
        },
        [addUrlParameterUpdate, keyPrefix, navigate]
    );

    const nextValue = getQueryObject(location.search)[keyPrefix] || defaultValue;

    // If the search filter has changed, replace the object reference.
    if (!isEqual(internalValue.current, nextValue)) {
        internalValue.current = nextValue;
    }

    return [internalValue.current, setValue];
}

export default useURLParameter;
