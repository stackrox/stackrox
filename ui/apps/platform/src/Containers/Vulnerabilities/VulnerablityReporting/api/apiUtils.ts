import { SearchFilter } from 'types/search';

export function getRequestQueryString(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .map(([key, val]) => `${key}:${Array.isArray(val) ? val.join(',') : val ?? ''}`)
        .join('+');
}
