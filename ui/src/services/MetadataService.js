import axios from 'axios/index';

/**
 * Fetches StackRox metadata.
 * @returns {Promise<Object, Error>} fulfilled with response
 */

// eslint-disable-next-line import/prefer-default-export
export function fetchMetadata() {
    const metadataUrl = '/v1/metadata';
    return axios.get(metadataUrl).then(response => ({
        response: response.data
    }));
}
