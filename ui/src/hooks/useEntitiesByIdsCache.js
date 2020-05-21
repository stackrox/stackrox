import { useReducer, useCallback } from 'react';

function reducer(state, payload) {
    if (state.length !== payload.entities.length) {
        return payload.entities;
    }

    // create a map {ID -> index in the array} for the cached entities
    // Note: we're doing it on every invocation of the reducer, though there is an opportunity
    // to avoid this extra computation by storing this derived data as well.
    const idToIndexMap = state.reduce((res, entity, index) => {
        const id = entity[payload.idAttribute];
        res[id] = index;
        return res;
    }, {});

    const isUpdateNeeded = payload.entities.some((entity, index) => {
        const id = entity[payload.idAttribute];
        if (idToIndexMap[id] === undefined) {
            // entity ID not found in the cached set
            return true;
        }
        if (payload.respectOrder && idToIndexMap[id] !== index) {
            // entity ID found but the order is different
            return true;
        }
        return false;
    });

    return isUpdateNeeded ? payload.entities : state;
}

const defaultOptions = {
    idAttribute: 'id',
    respectOrder: true,
};

/**
 * React Hook, similar to `useState`, which expects the value to be an array of immutable entities with unique IDs.
 * It'll not do anything to the stateful value in case update function is called with an array of entities, and
 * all those entities are already present in the current value. Entities are compared by ID attribute, therefore
 * it's expected that entities are not mutable (same ID means they're deep equal) and there are no duplicate IDs.
 *
 * @param {Object[]} [initialState=[]] cache will be initialized with this value (similar to `useState`)
 * @param {Object} [options] cache options
 * @param {string} [options.idAttribute=id] attribute name to use to compare entities
 * @param {boolean}[options.respectOrder=true] in case order of IDs is different from cached value, consider it an update
 * @return {[Object[], Function]} Returns a stateful value, and a function to update it (similar to `useState`)
 */
export default function useEntitiesByIdsCache(initialState = [], options = defaultOptions) {
    const { idAttribute, respectOrder } = { defaultOptions, ...options };
    const [cached, dispatch] = useReducer(reducer, initialState);

    // useCallback is needed, as some components might (and they're) have it as a dependency for useEffect
    // recreating function all the time will cause a rapid useEffect firing and eventual browser crash (oops)
    const updateCache = useCallback(
        (entities) => {
            dispatch({ entities, idAttribute, respectOrder });
        },
        [idAttribute, respectOrder]
    );

    return [cached, updateCache];
}
