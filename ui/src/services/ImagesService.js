import axios from 'axios';
import queryString from 'query-string';
import { normalize } from 'normalizr';
import searchOptionsToQuery from 'services/searchOptionsToQuery';

import { image as imageSchema } from './schemas';

const imagesUrl = '/v1/images';

/**
 * Fetches list of registered images.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of images (as defined in .proto)
 */
export function fetchImages(options) {
    const params = queryString.stringify({
        query: searchOptionsToQuery(options)
    });
    return axios
        .get(`${imagesUrl}?${params}`)
        .then(response => ({ response: normalize(response.data.images, [imageSchema]) }));
}

/**
 * Fetches image given an image SHA.
 *
 * @param {!Object} sha
 * @returns {Promise<?Object, Error>} fulfilled with object of image (as defined in .proto)
 */
export function fetchImage(sha) {
    if (!sha) throw new Error('Image SHA must be specified');
    return axios.get(`${imagesUrl}/${sha}`).then(response => {
        const image = Object.assign({}, response.data);
        const { name } = response.data;
        // this is to ensure that the single image api response merges with the slimmed table version properly
        if (name.sha) {
            image.sha = name.sha;
            image.name = name.fullName;
        }
        return { response: normalize(image, imageSchema) };
    });
}
