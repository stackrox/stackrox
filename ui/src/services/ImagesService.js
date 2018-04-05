import axios from 'axios';
import queryString from 'query-string';

/**
 * Fetches list of registered images.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of images (as defined in .proto)
 */
export default function fetchImages(filters) {
    const params = queryString.stringify({
        ...filters
    });
    const imagesUrl = '/v1/images';
    return axios.get(`${imagesUrl}?${params}`).then(response => ({
        response: response.data
    }));
}
