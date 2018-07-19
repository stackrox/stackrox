import axios from 'axios';
import queryString from 'query-string';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

const networkGraphUrl = '/v1/networkgraph';

/**
 * Fetches network graph nodes.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of nodes and edges (as defined in .proto)
 */
export default function fetchNetworkGraph(options) {
    const params = queryString.stringify({
        query: searchOptionsToQuery(options)
    });
    return axios
        .get(`${networkGraphUrl}?${params}`)
        .then(response => ({ response: response.data }));
}
