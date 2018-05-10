import axios from 'axios';
import queryString from 'query-string';

/**
 * Fetches list of registered risks.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of risks (as defined in .proto)
 */
export default function fetchDeployments(filters) {
    const params = queryString.stringify({
        ...filters
    });
    const deploymentsUrl = '/v1/deployments';
    return axios.get(`${deploymentsUrl}?${params}`).then(response => {
        const transformedData = response.data.deployments.map((o, index) => {
            const item = Object.assign({}, o);
            item.priority = index + 1;
            return item;
        });
        return { response: { deployments: transformedData } };
    });
}
