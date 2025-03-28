// Import from utils/searchUtils when it is written in TypeScript.
export type RestSearchOption = {
    label?: string | string[];
    type?: string; // for example, 'categoryOption'
    value: string | string[];
};

/**
 *  Convert array of search options to query string for sending to the server.
 */
export default function searchOptionsToQuery(searchOptions: RestSearchOption[]): string {
    return searchOptions
        .map((obj, i, { length }) => {
            const value = String(obj.value);
            if (obj.type) {
                return `${i !== 0 ? '+' : ''}${value}`;
            }
            return `${value}${i !== length - 1 ? ',' : ''}`;
        })
        .join('')
        .replace(/,\+/g, '+');
}
