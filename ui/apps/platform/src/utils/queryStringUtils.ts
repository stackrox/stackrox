import qs from 'qs';

export type BasePageAction = 'create' | 'edit';

export type ExtendedPageAction = BasePageAction | 'clone';

export function getQueryObject<T>(search: string): T {
    return qs.parse(search, { ignoreQueryPrefix: true }) as unknown as T;
}
