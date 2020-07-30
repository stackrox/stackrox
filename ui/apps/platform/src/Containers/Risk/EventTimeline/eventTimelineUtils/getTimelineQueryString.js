import queryService from 'utils/queryService';

/**
 * Returns the query string for a CSV export of an event timeline
 *
 * @param {object} exportParams - A collection of key/value pairs
 * @returns {string}
 */
function getTimelineQueryString(exportParams) {
    const queryValues = queryService.objectToWhereClause(exportParams, '&');
    const timelineQueryString = `query=${queryValues}`;

    return timelineQueryString;
}

export default getTimelineQueryString;
