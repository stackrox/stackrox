/**
 * Given the difference in milliseconds, this function will return an array of objects
 * with time units and their values
 * @param {number} differenceInMilliseconds - the milliseconds time difference
 */
const getTimeDiffWithUnit = (differenceInMilliseconds) => {
    const days = Math.floor(differenceInMilliseconds / (3600000 * 24));
    const hours = Math.floor(((differenceInMilliseconds / (3600000 * 24)) % 1) * 24);
    const minutes = Math.floor(((differenceInMilliseconds / 3600000) % 1) * 60);
    const seconds = Math.floor(((((differenceInMilliseconds / 3600000) % 1) * 60) % 1) * 60);
    if (days > 0) {
        return [{ timeDifference: days, unit: 'day', shortHandUnit: 'd' }];
    }
    if (hours > 0) {
        return [
            { timeDifference: hours, unit: 'hour', shortHandUnit: 'h' },
            { timeDifference: minutes, unit: 'minute', shortHandUnit: 'm' },
        ];
    }
    if (minutes > 0) {
        return [
            { timeDifference: minutes, unit: 'minute', shortHandUnit: 'm' },
            { timeDifference: seconds, unit: 'second', shortHandUnit: 's' },
        ];
    }
    if (seconds > 0) {
        return [{ timeDifference: seconds, unit: 'second', shortHandUnit: 's' }];
    }
    return [{ timeDifference: differenceInMilliseconds, unit: 'millisecond', shortHandUnit: 'ms' }];
};

export default getTimeDiffWithUnit;
