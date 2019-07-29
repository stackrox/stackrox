import axios from './instance';

const url = '/v1/risks';

/**
 * Fetches risk given subject ID and type
 *
 * @param {!string} subjectID
 * @param {!string} subjectType
 * @returns {Promise<Object, Error>} fulfilled with response
 */
function fetchRisk(subjectID, subjectType) {
    return axios.get(`${url}/${subjectType}/${subjectID}`).then(response => ({
        response: response.data
    }));
}

export default fetchRisk;
