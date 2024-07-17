import {
    CompoundSearchFilterConfig,
    deploymentSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageSearchFilterConfig,
} from '../types';
import {
    getEntities,
    getEntityAttributes,
    getDefaultEntity,
    getDefaultAttribute,
    makeFilterChipDescriptors,
} from './utils';

describe('utils', () => {
    describe('getEntities', () => {
        it('should get the entities in a config object', () => {
            const config: Partial<CompoundSearchFilterConfig> = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                'Image CVE': imageCVESearchFilterConfig,
            };

            const result = getEntities(config);

            expect(result).toStrictEqual(['Image', 'Deployment', 'Image CVE']);
        });
    });

    describe('getEntityAttributes', () => {
        it('should get the attributes of an entity in a config object', () => {
            const config: Partial<CompoundSearchFilterConfig> = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                'Image CVE': imageCVESearchFilterConfig,
            };

            const result = getEntityAttributes('Image CVE', config);

            expect(result).toStrictEqual([
                {
                    displayName: 'Name',
                    filterChipLabel: 'Image CVE',
                    searchTerm: 'CVE',
                    inputType: 'autocomplete',
                },
                {
                    displayName: 'Discovered time',
                    filterChipLabel: 'Image CVE discovered time',
                    searchTerm: 'CVE Created Time',
                    inputType: 'date-picker',
                },
                {
                    displayName: 'CVSS',
                    filterChipLabel: 'CVSS',
                    searchTerm: 'CVSS',
                    inputType: 'condition-number',
                },
            ]);
        });
    });

    describe('getDefaultEntity', () => {
        it('should get the default (first) entity in a config object', () => {
            const config: Partial<CompoundSearchFilterConfig> = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                'Image CVE': imageCVESearchFilterConfig,
            };

            const result = getDefaultEntity(config);

            expect(result).toStrictEqual('Image');
        });

        // @TODO: Worth considering if we want to ignore vs. highlight the issue
        it("should ignore entity names that aren't valid", () => {
            const config = {
                BOGUS: {
                    hello: 'friend',
                },
                Image: imageSearchFilterConfig,
            };

            const result = getDefaultEntity(config);

            expect(result).toStrictEqual('Image');
        });
    });

    describe('getDefaultAttribute', () => {
        it('should get the default (first) attribute of a specific entity in a config object', () => {
            const config: Partial<CompoundSearchFilterConfig> = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                'Image CVE': imageCVESearchFilterConfig,
            };

            const result = getDefaultAttribute('Image CVE', config);

            expect(result).toStrictEqual('Name');
        });
    });

    describe('makeFilterChipDescriptors', () => {
        it('should create an array of FilterChipGroupDescriptor objects from a config object', () => {
            const config: Partial<CompoundSearchFilterConfig> = {
                'Image CVE': imageCVESearchFilterConfig,
            };

            const result = makeFilterChipDescriptors(config);

            expect(result).toStrictEqual([
                {
                    displayName: 'Image CVE',
                    searchFilterName: 'CVE',
                },
                {
                    displayName: 'Image CVE discovered time',
                    searchFilterName: 'CVE Created Time',
                },
                {
                    displayName: 'CVSS',
                    searchFilterName: 'CVSS',
                },
            ]);
        });
    });
});
