import isPlainObject from 'lodash/isPlainObject';
import qs, { ParsedQs } from 'qs';

import { SearchFilter } from 'types/search';

export type PageAction = 'clone' | 'create' | 'edit';

function isValidAction(action: unknown): action is PageAction {
    return action === 'clone' || action === 'create' || action === 'edit';
}

function isParsedQs(s: unknown): s is ParsedQs {
    return isPlainObject(s);
}

function isValidFilterValue(value: unknown): value is string | string[] {
    if (typeof value === 'string') {
        return true;
    }

    if (Array.isArray(value) && value.every((item) => typeof item === 'string')) {
        return true;
    }

    return false;
}

function isValidFilter(s: unknown): s is SearchFilter {
    return isParsedQs(s) && Object.values(s).every((value) => isValidFilterValue(value));
}

export type PoliciesSearch = {
    pageAction?: PageAction;
    searchFilter?: SearchFilter;
};

/*
 * Given search query string from location, return validated action string and filter object.
 *
 * Examples of search query string:
 * ?action=create
 * ?action=edit
 * ?s[Lifecycle Stage]=BUILD
 * ?s[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY
 * ?s[Lifecycle State]=RUNTIME&s[Severity]=CRITICAL_SEVERITY
 */
export function parsePoliciesSearchString(search: string): PoliciesSearch {
    const { action, s } = qs.parse(search, { ignoreQueryPrefix: true });

    return {
        pageAction: isValidAction(action) ? action : undefined,
        searchFilter: isValidFilter(s) ? s : undefined,
    };
}

export function getSearchStringForFilter(s: SearchFilter): string {
    return qs.stringify(
        { s },
        {
            arrayFormat: 'repeat',
            encodeValuesOnly: true,
        }
    );
}

/*
 * Return request query string for search filter. Omit filter criterion:
 * If option does not have value.
 */
export function getRequestQueryStringForSearchFilter(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .filter(([, value]) => value.length !== 0)
        .map(([key, value]) => `${key}:${Array.isArray(value) ? value.join(',') : value}`)
        .join('+');
}
