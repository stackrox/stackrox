export function truncate(str, maxLength = 200) {
    if (str.length <= maxLength) {
        return str;
    }

    const stringToTrim = str.substr(0, maxLength + 1);

    const truncatedStr = stringToTrim.substr(
        0,
        Math.min(stringToTrim.length, stringToTrim.lastIndexOf(' ', maxLength + 1))
    );
    return `${truncatedStr}…`;
}

export function middleTruncate(str, maxLength = 22) {
    if (str.length <= maxLength) {
        return str;
    }
    const sideLength = maxLength / 2;
    const left = str.substr(0, sideLength);
    const right = str.substr(str.length - sideLength, str.length);
    const ellipsis = '\u2026';
    return `${left}${ellipsis}${right}`;
}

export function pluralizeHas(len) {
    return len === 1 ? 'has' : 'have';
}

export default {
    truncate,
    pluralizeHas,
};
