/**
 * Determines the width of the number-of-events container that displays in the top right corner
 * of a clustered event
 * @param {number} numEvents - The number of clustered events
 * @returns {number} width of the container
 */
export function getNumEventsBackgroundWidth(numEvents) {
    if (numEvents <= 9) return 9;
    return 13;
}

/**
 * Determines the text (number of events) for the container that displays in the
 * top right corner of a clustered event
 * @param {number} numEvents -  The number of clustered events
 * @returns {string} the text displayed in the container
 */
export function getNumEventsText(numEvents) {
    if (numEvents <= 9) return String(numEvents);
    return '9+';
}
