import { getLegacySearchFilterAttributes, searchFilterConfig } from './searchFilterConfig';

describe('getLegacySearchFilterAttributes', () => {
    it('returns a removable text attribute for a key not represented in the config', () => {
        const attributes = getLegacySearchFilterAttributes(
            { 'Image Tag': 'latest' },
            searchFilterConfig
        );

        expect(attributes).toEqual([
            {
                displayName: 'Image Tag',
                filterChipLabel: 'Image Tag',
                searchTerm: 'Image Tag',
                inputType: 'text',
            },
        ]);
    });

    it('excludes keys owned by the scope selectors (Cluster/Namespace/Deployment)', () => {
        const attributes = getLegacySearchFilterAttributes(
            { Cluster: 'production', Namespace: ['stackrox'], Deployment: 'central' },
            searchFilterConfig
        );

        expect(attributes).toEqual([]);
    });

    it('retains a case-different scope key, which useScopeHierarchy does not consume', () => {
        // useScopeHierarchy reads scope keys by exact property access (searchFilter.Cluster), so a
        // lowercase "cluster" from a legacy/bookmarked URL is not owned by the scope selector and
        // must stay as a removable chip rather than becoming a hidden, unremovable filter.
        const attributes = getLegacySearchFilterAttributes(
            { cluster: 'production' },
            searchFilterConfig
        );

        expect(attributes).toEqual([
            {
                displayName: 'cluster',
                filterChipLabel: 'cluster',
                searchTerm: 'cluster',
                inputType: 'text',
            },
        ]);
    });

    it('excludes keys already covered by a config attribute, matching case-insensitively', () => {
        // "Deployment Label" is a configured attribute; its lowercase variant must not be duplicated.
        const attributes = getLegacySearchFilterAttributes(
            { 'deployment label': ['app=web'] },
            searchFilterConfig
        );

        expect(attributes).toEqual([]);
    });

    it('skips keys with empty values so no empty chip group is rendered', () => {
        const attributes = getLegacySearchFilterAttributes(
            { 'Image Tag': '', 'Service Account': [] },
            searchFilterConfig
        );

        expect(attributes).toEqual([]);
    });

    it('retains keys with array values and returns only the uncovered ones', () => {
        const attributes = getLegacySearchFilterAttributes(
            { Cluster: 'production', 'Service Account': ['default', 'admin'] },
            searchFilterConfig
        );

        expect(attributes).toEqual([
            {
                displayName: 'Service Account',
                filterChipLabel: 'Service Account',
                searchTerm: 'Service Account',
                inputType: 'text',
            },
        ]);
    });
});
