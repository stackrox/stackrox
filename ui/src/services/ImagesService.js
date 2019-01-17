import axios from 'axios';
import queryString from 'qs';
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
    const params = queryString.stringify(
        { query: searchOptionsToQuery(options) },
        { encode: false, arrayFormat: 'repeat' }
    );
    return axios
        .get(`${imagesUrl}?${params}`)
        .then(response => ({ response: normalize(response.data.images, [imageSchema]) }));
}

/**
 * Fetches image given an image ID.
 *
 * @param {!Object} id
 * @returns {Promise<?Object, Error>} fulfilled with object of image (as defined in .proto)
 */
export function fetchImage(id) {
    if (!id) throw new Error('Image ID must be specified');
    return axios.get(`${imagesUrl}/${id}`).then(response => {
        const image = Object.assign({}, response.data);
        const { name } = response.data;
        image.name = name.fullName;
        return { response: normalize(image, imageSchema) };
    });
}
