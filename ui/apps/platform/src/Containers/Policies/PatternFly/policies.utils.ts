import isPlainObject from 'lodash/isPlainObject';
import qs, { ParsedQs } from 'qs';

export type PoliciesAction = 'create' | 'edit';

/*
 * Examples of policy filter object properties parsed from search query string:
 * 'Lifecycle Stage': 'BUILD' from 's[Lifecycle Stage]=BUILD
 * 'Lifecycle Stage': ['BUILD', 'DEPLOY'] from 's[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY'
 */
export type PoliciesFilter = Record<string, string | string[]>;

function isValidAction(action: unknown): action is PoliciesAction {
    return action === 'create' || action === 'edit';
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

function isValidFilter(s: unknown): s is PoliciesFilter {
    return isParsedQs(s) && Object.values(s).every((value) => isValidFilterValue(value));
}

export type PoliciesSearch = {
    action?: PoliciesAction;
    filter?: PoliciesFilter;
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
        action: isValidAction(action) ? action : undefined,
        filter: isValidFilter(s) ? s : undefined,
    };
}
