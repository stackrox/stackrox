export function truncate(str: string, maxLength = 200) {
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

export function pluralizeHas(len) {
    return len === 1 ? 'has' : 'have';
}

export default {
    truncate,
    pluralizeHas,
};
