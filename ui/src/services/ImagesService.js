import queryString from 'qs';
import { normalize } from 'normalizr';
import searchOptionsToQuery from 'services/searchOptionsToQuery';
import axios from './instance';

import { image as imageSchema } from './schemas';

const imagesUrl = '/v1/images';
const imagesCountUrl = '/v1/imagescount';

/**
 * Fetches list of registered images.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of images (as defined in .proto)
 */
export function fetchImagesById(options) {
    const params = queryString.stringify(
        { query: searchOptionsToQuery(options) },
        { arrayFormat: 'repeat' }
    );
    return axios
        .get(`${imagesUrl}?${params}`)
        .then((response) => ({ response: normalize(response.data.images, [imageSchema]) }));
}

/**
 * Fetches list of registered images, using the input hooks to give the results.
 *
 * @returns {Promise<Object[], Error>} fulfilled with array of images (as defined in .proto)
 */
export function fetchImages(options, sortOption, page, pageSize) {
    const offset = page * pageSize;
    const query = searchOptionsToQuery(options);
    const params = queryString.stringify(
        {
            query,
            pagination: {
                offset,
                limit: pageSize,
                sortOption,
            },
        },
        { arrayFormat: 'repeat', allowDots: true }
    );
    return axios
        .get(`${imagesUrl}?${params}`)
        .then((response) => ({ response: normalize(response.data.images, [imageSchema]) }))
        .then((obj) => {
            if (obj.response.entities.image === undefined) {
                return [];
            }
            return Object.values(obj.response.entities.image);
        });
}

/**
 * Fetches list of count of images, using the input hooks to give the results.
 *
 * @returns Nothing. Responds through hooks.
 */
export function fetchImageCount(options) {
    const params = queryString.stringify(
        { query: searchOptionsToQuery(options) },
        { arrayFormat: 'repeat' }
    );
    return axios.get(`${imagesCountUrl}?${params}`).then((response) => response.data.count);
}

/**
 * Fetches a specified image.
 *
 * @returns {Promise<?Object, Error>} fulfilled with object of image (as defined in .proto)
 */
export function fetchImage(id) {
    if (!id) throw new Error('Image ID must be specified');
    return axios.get(`${imagesUrl}/${id}`).then((response) => {
        const image = { ...response.data };
        const { name } = response.data;
        image.name = name.fullName;
        return image;
    });
}
