import { useReducer, useCallback } from 'react';

function reducer(state, payload) {
    if (state.length !== payload.entities.length) {
        return payload.entities;
    }
    const currentIds = state.reduce(
        (set, entity) => set.add(entity[payload.idAttribute]),
        new Set()
    );
    const anyNewEntities = payload.entities.some(
        entity => !currentIds.has(entity[payload.idAttribute])
    );
    if (anyNewEntities) {
        return payload.entities;
    }
    return state;
}

/**
 * React Hook, similar to `useState`, which expects the value to be an array of immutable entities with unique IDs.
 * It'll not do anything to the stateful value in case update function is called with an array of entities, and
 * all those entities are already present in the current value. Entities are compared by ID attribute, therefore
 * it's expected that entities are not mutable (same ID means they're deep equal).
 *
 * @param {Object[]} [initialState=[]] cache will be initialized with this value (similar to `useState`)
 * @param {string} [idAttribute=id] attribute name to use to compare entities
 * @return {[Object[], Function]} Returns a stateful value, and a function to update it (similar to `useState`)
 */
export default function useEntitiesByIdsCache(initialState = [], idAttribute = 'id') {
    const [cached, dispatch] = useReducer(reducer, initialState);

    // useCallback is needed, as some components might (and they're) have it as a dependency for useEffect
    // recreating function all the time will cause a rapid useEffect firing and eventual browser crash (oops)
    const updateCache = useCallback(
        entities => {
            dispatch({ entities, idAttribute });
        },
        [idAttribute]
    );

    return [cached, updateCache];
}
