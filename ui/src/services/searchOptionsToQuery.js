/**
 *  Converts list of search options to a query for sending to the server.
 *
 *  @param {!Object[]} searchOptions an array of search options
 *  @returns {string} search query string
 */
export default function searchOptionsToQuery(searchOptions) {
    return searchOptions
        .map((obj, i, { length }) => {
            if (obj.type) return `${i !== 0 ? '+' : ''}${obj.value}`;
            return `${obj.value}${i !== length - 1 ? ',' : ''}`;
        })
        .join('')
        .replace(',+', '+');
}
