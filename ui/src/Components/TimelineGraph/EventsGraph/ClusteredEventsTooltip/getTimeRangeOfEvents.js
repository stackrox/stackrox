const getTimeRangeOfEvents = (events) => {
    const { min, max } = events.reduce(
        (acc, curr) => {
            let newMin = acc.min;
            let newMax = acc.max;
            if (curr.differenceInMilliseconds < acc.min) {
                newMin = curr.differenceInMilliseconds;
            }
            if (curr.differenceInMilliseconds > acc.max) {
                newMax = curr.differenceInMilliseconds;
            }
            return {
                min: newMin,
                max: newMax,
            };
        },
        { min: Infinity, max: -Infinity }
    );
    return {
        timeRangeOfEvents: max - min,
        unit: 'ms',
    };
};

export default getTimeRangeOfEvents;
