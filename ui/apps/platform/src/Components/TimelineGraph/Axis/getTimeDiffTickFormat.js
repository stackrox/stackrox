import getTimeDiffWithUnit from 'Components/TimelineGraph/timelineGraphUtils/getTimeDiffWithUnit';

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
        const timeDiffWithUnit = getTimeDiffWithUnit(milliseconds);
        return timeDiffWithUnit.reduce((acc, curr, currIndex, srcArray) => {
            let newValue = acc;
            if (currIndex === 0) {
                newValue = `${newValue}${curr.timeDifference}`;
            } else {
                newValue = `${newValue}:${String(curr.timeDifference).padStart(2, '0')}`;
            }
            if (currIndex === srcArray.length - 1) {
                return `${newValue}${srcArray[0].shortHandUnit}`;
            }
            return newValue;
        }, '+');
    }
    return null;
}

export default getTimeDiffTickFormat;
