import isEqual from 'lodash/isEqual';
import merge from 'lodash/merge';

/**
 * Given the map of existing entities by their IDs and a map of updated entities (e.g. received from the server),
 * deeply merges them using copy-on-change approach (i.e. applying only to the entity objects that got updated).
 * This is a pure function that doesn't mutate its arguments.
 *
 * @param {!Object.<string, Object>} existingEntitiesById map of "id -> entity" of existing entities
 * @param {!Object.<string, Object>} newEntitiesById map of "id -> entity" of potentially updated entities
 * @returns {Object.<string, Object>} map of "id -> entity" with updated entities deeply merged in
 */
export default function mergeEntitiesById(existingEntitiesById, newEntitiesById) {
    return Object.keys(newEntitiesById).reduce((result, id) => {
        if (!existingEntitiesById[id]) return { ...result, [id]: newEntitiesById[id] };
        if (isEqual(existingEntitiesById[id], newEntitiesById[id])) return result;
        return {
            ...result,
            [id]: merge({}, existingEntitiesById[id], newEntitiesById[id])
        };
    }, existingEntitiesById);
}
