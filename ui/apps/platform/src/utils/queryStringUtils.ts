import qs from 'qs';
import isPlainObject from 'lodash/isPlainObject';

export type BasePageAction = 'create' | 'edit';

export type ExtendedPageAction = BasePageAction | 'clone' | 'generate';

export function getQueryObject(search: string): qs.ParsedQs {
    return qs.parse(search, { ignoreQueryPrefix: true, arrayLimit: 200 });
}

export function getQueryString<T>(
    searchObject: T,
    options?: qs.IStringifyOptions<qs.BooleanOptional>
): string {
    const allOptions = { encode: false, addQueryPrefix: true, ...options };
    return qs.stringify(searchObject, allOptions);
}

export function isParsedQs(s: unknown): s is qs.ParsedQs {
    return isPlainObject(s);
}
