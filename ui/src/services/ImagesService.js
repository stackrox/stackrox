import axios from 'axios';
import queryString from 'query-string';
import reduce from 'lodash/reduce';

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
    return axios.get(`${imagesUrl}?${params}`).then(response => {
        const transformedImages = Object.assign({}, response.data);
        transformedImages.images.map(image => {
            const o = image;
            o.scanComponentsLength = o && o.scan && o.scan.components.length;
            o.scanComponentsSum =
                o &&
                o.scan &&
                reduce(o.scan.components, (sum, component) => sum + component.vulns.length, 0);
            return o;
        });
        return {
            response: transformedImages
        };
    });
}
