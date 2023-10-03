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

export function dedupeDelimitedString(value: string, delimiter = ','): string[] {
    return Array.from(new Set(value.split(delimiter).map((v) => v.trim())));
}
