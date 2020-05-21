export function getHighValue(data, keyToCheck, multiple = 0, shouldPad = false) {
    const max = data.reduce(
        (currentHighest, item) =>
            item[keyToCheck] > currentHighest ? item[keyToCheck] : currentHighest,
        -Infinity
    );

    if (multiple) {
        const nextHighestMultiple = Math.ceil(max / multiple) * multiple;

        if (shouldPad) {
            return nextHighestMultiple + multiple;
        }

        return nextHighestMultiple;
    }
    return max;
}

export function getLowValue(data, keyToCheck, multiple = 0) {
    const min = data.reduce(
        (currentLowest, item) =>
            item[keyToCheck] < currentLowest ? item[keyToCheck] : currentLowest,
        Infinity
    );

    if (multiple) {
        const nextLowestMultiple = Math.floor(min / multiple) * multiple;

        return nextLowestMultiple;
    }
    return min;
}

export default {
    getHighValue,
    getLowValue,
};
