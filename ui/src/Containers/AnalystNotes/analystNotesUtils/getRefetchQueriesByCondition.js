/**
 * @typedef {Object} Result
 * @property {function} refetchQueries - The function that returns the array of queries and their variables (ie. () => [{ query, variables }])
 * @property {booolean} awaitRefetchQueries - Determines if the mutation should wait until the refetch queries resolve, before it resolves
 */

/**
 * Gets the queries (with their variables) that need to be refetched once a mutation occurs
 * @param {Object[]} queries - The array of refetchQueries with the added field of "exclude", which determines whether to exclude a query
 * @returns {Result}
 * 
 * Example: getRefetchQueriesByCondition([
        { query: QUERY_IS_REFETCHED, variables, exclude: false },
        { query: QUERY_IS_NOT_REFETCHED, variables, exclude: true }
    ])
 */
function getRefetchQueriesByCondition(queries) {
    const refetchQueries = queries.filter(query => !query.exclude);
    return {
        refetchQueries: () => refetchQueries,
        awaitRefetchQueries: true
    };
}

export default getRefetchQueriesByCondition;
