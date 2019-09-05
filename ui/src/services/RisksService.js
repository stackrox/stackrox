import axios from './instance';

const url = '/v1/risks';

/**
 * Fetches risk given entity ID and type
 *
 * @param {!string} entityID
 * @param {!string} entityType
 * @returns {Promise<Object, Error>} fulfilled with response
 */
function fetchRisk(entityID, entityType) {
    return axios.get(`${url}/${entityType}/${entityID}`).then(response => response.data);
}

export default fetchRisk;
