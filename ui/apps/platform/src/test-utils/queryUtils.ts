import { MatcherFunction } from '@testing-library/react';

/**
 * Checks that the provided element matches the provided term and that it _does not_ contain
 * a child element that matches the provided term. This allows querying for the `textContent`
 * of an element without also returning every descendent element of the match.
 */
function isDeepestElementMatch<TermType>(
    element: Element | null,
    term: TermType,
    matcher: (element: Element, term: TermType) => boolean
) {
    if (!element || !matcher(element, term)) {
        return false;
    }
    const children = Array.from(element.children);
    for (let i = 0; i < children.length; i += 1) {
        if (matcher(children[i], term)) {
            return false;
        }
    }
    return true;
}

const stringMatcher = (element: Element, term: string) => !!element?.textContent?.includes(term);
const regexMatcher = (element: Element, term: RegExp) => term.test(element?.textContent ?? '');

/**
 * Creates a MatcherFunction https://testing-library.com/docs/queries/about#textmatch that searches
 * for the deepest element with a `textContent` that matches the provided term. Useful to match
 * text that may be separated by multiple HTML elements.
 *
 * @param term The search term to match
 * @return A MatcherFunction to run against the document tree
 */
export function withTextContent(term: string | RegExp): MatcherFunction {
    return typeof term === 'string'
        ? (_content, element) => isDeepestElementMatch(element, term, stringMatcher)
        : (_content, element) => isDeepestElementMatch(element, term, regexMatcher);
}
