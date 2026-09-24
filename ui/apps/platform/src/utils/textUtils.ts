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

/**
 * Uppercases the first character of a string, leaving the remainder verbatim.
 * Unlike lodash `capitalize`, it does not lower-case the rest, so names, quotes,
 * and acronyms in the remainder are preserved (e.g. the cluster name in
 * "no connection to cluster ..."). A nullish or empty input yields an empty
 * string, matching lodash `upperFirst`.
 */
export function capitalizeFirst(str: string | null | undefined): string {
    if (!str) {
        return '';
    }
    return str.charAt(0).toUpperCase() + str.slice(1);
}

export function dedupeDelimitedString(value: string, delimiter = ','): string[] {
    return Array.from(new Set(value.split(delimiter).map((v) => v.trim())));
}

/**
 * Returns the first item of the list if there's only one item, or the count of items with a label if there are multiple items.
 * @param {string[]} items - The list of items
 * @param {string} multipleItemLabel - The label to display when there are multiple items
 * @returns {string} - The first item of the list or the count of items with the provided label
 */
export function displayOnlyItemOrItemCount(items: string[], multipleItemLabel: string): string {
    if (items.length > 1) {
        return `${items.length} ${multipleItemLabel}`;
    }
    return items[0];
}
