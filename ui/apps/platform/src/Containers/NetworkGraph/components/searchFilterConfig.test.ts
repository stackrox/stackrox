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
