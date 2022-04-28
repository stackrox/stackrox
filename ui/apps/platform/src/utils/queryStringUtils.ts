import qs from 'qs';
import isPlainObject from 'lodash/isPlainObject';

export type BasePageAction = 'create' | 'edit';

export type ExtendedPageAction = BasePageAction | 'clone' | 'generate';

export function getQueryObject(search: string): qs.ParsedQs {
    return qs.parse(search, { ignoreQueryPrefix: true });
}

export function getQueryString<T>(searchObject: T): string {
    return qs.stringify(searchObject, { encode: false, addQueryPrefix: true });
}

export function isParsedQs(s: unknown): s is qs.ParsedQs {
    return isPlainObject(s);
}
