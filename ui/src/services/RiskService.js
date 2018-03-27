import axios from 'axios';

/**
 * Fetches list of registered risks.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of risks (as defined in .proto)
 */
export default function fetchDeployments() {
    const deploymentsUrl = '/v1/deployments';
    return axios.get(deploymentsUrl).then(response => ({
        response: response.data
    }));
}
