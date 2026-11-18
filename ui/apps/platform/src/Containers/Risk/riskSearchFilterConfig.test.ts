import { isNumericSearchTerm } from 'utils/searchUtils';

import { getLegacySearchFilterAttributes, searchFilterConfig } from './riskSearchFilterConfig';

const allAttributes = searchFilterConfig.flatMap((entity) => entity.attributes);
const allSearchTerms = allAttributes.map((attribute) => attribute.searchTerm);

describe('riskSearchFilterConfig', () => {
    it('exposes the 13 PM-approved curated fields', () => {
        expect(allSearchTerms).toHaveLength(13);
        expect(new Set(allSearchTerms)).toEqual(
            new Set([
                'Deployment',
                'Deployment Label',
                'Deployment Annotation',
                'Deployment Type',
                'Container Type',
                'Privileged',
                'Read Only Root Filesystem',
                'Service Account',
                'Secret',
                'Exposure Level',
                'Port',
                'Cluster',
                'Namespace',
            ])
        );
    });

    it('scopes every entity to the DEPLOYMENTS search category for autocomplete', () => {
        searchFilterConfig.forEach((entity) => {
            expect(entity.searchCategory).toBe('DEPLOYMENTS');
        });
    });

    it('registers the numeric Port field so autocomplete omits unsupported modifiers', () => {
        // Port offers autocomplete suggestions, but must be treated as numeric so the shared
        // autocomplete does not add the regex/exact-match modifiers the search API rejects for
        // numeric fields ("modifiers not supported for numeric query: [Equality]").
        const portAttribute = allAttributes.find((attribute) => attribute.searchTerm === 'Port');
        expect(portAttribute?.inputType).toBe('autocomplete');
        expect(isNumericSearchTerm('Port')).toBe(true);
    });

    it('does not include dropped long-tail or workflow-view fields', () => {
        expect(allSearchTerms).not.toContain('CPU Cores Limit');
        expect(allSearchTerms).not.toContain('Platform Component');
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

    it('ignores workflow-view keys and empty values', () => {
        const result = getLegacySearchFilterAttributes({
            'Platform Component': 'true',
            Secret: '',
            'Orchestrator Component': 'true',
        });
        expect(result).toEqual([]);
    });

    it('matches config keys case-insensitively', () => {
        // The label component reads values case-insensitively, so a differently-cased key that the
        // config already handles must not also render as a separate legacy chip.
        const result = getLegacySearchFilterAttributes({ deployment: 'nginx' });
        expect(result).toEqual([]);
    });
});
