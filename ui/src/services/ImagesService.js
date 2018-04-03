import axios from 'axios';

/**
 * Fetches list of registered images.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of images (as defined in .proto)
 */
export default function fetchImages() {
    const imagesUrl = '/v1/images';
    return axios.get(imagesUrl).then(response => ({
        response: response.data
    }));
}
