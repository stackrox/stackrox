import axios from 'axios';
import { normalize } from 'normalizr';
import queryString from 'query-string';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

import { secret as secretSchema } from './schemas';

const secretsUrl = '/v1/secrets';

/**
 * Fetches list of secrets.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of secrets (as defined in .proto)
 */
export function fetchSecrets(options) {
    const params = queryString.stringify({
        query: searchOptionsToQuery(options)
    });
    return axios.get(`${secretsUrl}?${params}`).then(response => ({
        response: normalize(response.data.secrets, [secretSchema])
    }));
}

export function fetchSecret(id) {
    if (!id) throw new Error('Secret ID must be specified');
    return axios
        .get(`${secretsUrl}/${id}`)
        .then(response => ({ response: normalize(response.data, secretSchema) }));
}
