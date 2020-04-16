import pluralize from 'pluralize';
/**
 * Formats the milliseconds value to the format +N:Mx, where x represents
 * the time measurements (ms for milliseconds, s for seconds, m for minutes, h for hours) and
 * N and M represent numbers
 * @param {Number} milliseconds - The GraphQL query/mutation errors
 * @param {Number} i - The current tick index
 * @param {Number} values - The tick values
 * @returns {String} - The formatted time difference
 */
function getTimeDiffTickFormat(milliseconds, i, values) {
    if (i !== 0 && i !== values.length - 1) {
        const days = Math.floor(milliseconds / (3600000 * 24));
        const hours = Math.floor(((milliseconds / (3600000 * 24)) % 1) * 24);
        const minutes = Math.floor(((milliseconds / 3600000) % 1) * 60);
        const seconds = Math.floor(((((milliseconds / 3600000) % 1) * 60) % 1) * 60);
        if (days > 0) return `+${days}${pluralize('day', days)}`;
        if (hours > 0) return `+${hours}:${String(minutes).padStart(2, '0')}h`;
        if (minutes > 0) return `+${minutes}:${String(seconds).padStart(2, '0')}m`;
        if (seconds > 0) return `+${seconds}s`;
        return `+${milliseconds}ms`;
    }
    return null;
}

export default getTimeDiffTickFormat;
