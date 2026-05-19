import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { Flex, FlexItem } from '@patternfly/react-core';

import type { SearchResult } from 'services/SearchService';
import { safeGeneratePath } from 'utils/urlUtils';

import NotApplicable from './NotApplicable';
import type { SearchResultCategoryMap } from './searchCategories';

type ViewLinksProps = {
    searchResult: SearchResult & {
        locationTextForCategory: string;
    };
    searchResultCategoryMap: SearchResultCategoryMap;
};

export function resolveParams(template: string, params: Partial<Record<string, unknown>>): string {
    return template.replace(/:(\w+)/g, (match, key) => {
        const value = params[key];
        // We cannot easily restrict the type of the value to a string without a large refactor, so we
        // disable the rule and rely on what testing we have for now...
        // eslint-disable-next-line @typescript-eslint/no-base-to-string
        return value != null ? String(value) : match;
    });
}

export function buildViewLinkUrl(
    basePath: string,
    searchResult: Partial<Record<string, unknown>>,
    searchParams: string | undefined
): string {
    const queryIndex = basePath.indexOf('?');
    const pathPattern = queryIndex >= 0 ? basePath.slice(0, queryIndex) : basePath;
    const baseQuery = queryIndex >= 0 ? basePath.slice(queryIndex + 1) : undefined;

    const path = safeGeneratePath(pathPattern, searchResult, pathPattern);
    const resolvedBaseQuery = baseQuery ? resolveParams(baseQuery, searchResult) : undefined;
    const queryParts = [resolvedBaseQuery, searchParams].filter(Boolean).join('&');

    return queryParts ? `${path}?${queryParts}` : path;
}

function ViewLinks({ searchResult, searchResultCategoryMap }: ViewLinksProps): ReactElement {
    const { viewLinks } = searchResultCategoryMap[searchResult.category] ?? {};

    if (viewLinks?.length) {
        return (
            <Flex spaceItems={{ default: 'spaceItemsMd' }}>
                {viewLinks.map(({ basePath, linkText, searchParams }) => (
                    <FlexItem key={linkText}>
                        <Link
                            to={buildViewLinkUrl(basePath, searchResult, searchParams)}
                            className="pf-v6-u-text-nowrap"
                        >
                            {linkText}
                        </Link>
                    </FlexItem>
                ))}
            </Flex>
        );
    }

    return <NotApplicable />;
}

export default ViewLinks;
