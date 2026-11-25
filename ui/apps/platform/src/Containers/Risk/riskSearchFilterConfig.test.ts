import { isNumericSearchTerm } from 'utils/searchUtils';

import { getLegacySearchFilterAttributes, searchFilterConfig } from './riskSearchFilterConfig';

const allAttributes = searchFilterConfig.flatMap((entity) => entity.attributes);

describe('riskSearchFilterConfig', () => {
    // Only the cross-module contract is asserted here; re-checking the curated set against the same
    // config it is read from would just mirror the implementation.
    it('registers the numeric Port field so autocomplete omits unsupported modifiers', () => {
        // Port offers autocomplete suggestions, but must be treated as numeric so the shared
        // autocomplete does not add the regex/exact-match modifiers the search API rejects for
        // numeric fields ("modifiers not supported for numeric query: [Equality]").
        const portAttribute = allAttributes.find((attribute) => attribute.searchTerm === 'Port');
        expect(portAttribute?.inputType).toBe('autocomplete');
        expect(isNumericSearchTerm('Port')).toBe(true);
    });
});

describe('getLegacySearchFilterAttributes', () => {
    it('returns nothing when every active filter key is in the config', () => {
        const result = getLegacySearchFilterAttributes({ Deployment: 'nginx', Cluster: 'remote' });
        expect(result).toEqual([]);
    });

    it('surfaces a dropped field from a bookmarked URL as a removable text chip', () => {
        const result = getLegacySearchFilterAttributes({ 'CPU Cores Limit': '2' });
        expect(result).toEqual([
            {
                displayName: 'CPU Cores Limit',
                filterChipLabel: 'CPU Cores Limit',
                searchTerm: 'CPU Cores Limit',
                inputType: 'text',
            },
        ]);
    });

    it('ignores empty values', () => {
        const result = getLegacySearchFilterAttributes({ 'CPU Cores Limit': '' });
        expect(result).toEqual([]);
    });

    it('surfaces a workflow-view key as a removable chip when it leaks into the URL filter', () => {
        // "Platform Component" is normally owned by the FilteredWorkflowView selector, but if a
        // crafted or legacy URL puts it in the URL-owned filter it still scopes the deployments
        // query (notably in Full view). It must be a visible, removable chip, not a hidden filter.
        const result = getLegacySearchFilterAttributes({ 'Platform Component': 'true' });
        expect(result).toEqual([
            {
                displayName: 'Platform Component',
                filterChipLabel: 'Platform Component',
                searchTerm: 'Platform Component',
                inputType: 'text',
            },
        ]);
    });

    it('matches config keys case-insensitively', () => {
        // The label component reads values case-insensitively, so a differently-cased key that the
        // config already handles must not also render as a separate legacy chip.
        const result = getLegacySearchFilterAttributes({ deployment: 'nginx' });
        expect(result).toEqual([]);
    });
});
