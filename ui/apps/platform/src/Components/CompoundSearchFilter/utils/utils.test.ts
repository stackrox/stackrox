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
                    displayName: 'ID',
                    filterChipLabel: 'Image CVE ID',
                    searchTerm: 'CVE ID',
                    inputType: 'autocomplete',
                },
                {
                    displayName: 'Discovered Time',
                    filterChipLabel: 'Image CVE Discovered Time',
                    searchTerm: 'CVE Created Time',
                    inputType: 'date-picker',
                },
                {
                    displayName: 'CVSS',
                    filterChipLabel: 'Image CVE CVSS',
                    searchTerm: 'CVSS',
                    inputType: 'condition-number',
                },
                {
                    displayName: 'Type',
                    filterChipLabel: 'Image CVE Type',
                    searchTerm: 'CVE Type',
                    inputType: 'select',
                    inputProps: {
                        options: [{ label: 'Image CVE', value: 'IMAGE_CVE' }],
                    },
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

            expect(result).toStrictEqual('ID');
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
                    displayName: 'Image CVE ID',
                    searchFilterName: 'CVE ID',
                },
                {
                    displayName: 'Image CVE Discovered Time',
                    searchFilterName: 'CVE Created Time',
                },
                {
                    displayName: 'Image CVE CVSS',
                    searchFilterName: 'CVSS',
                },
                {
                    displayName: 'Image CVE Type',
                    searchFilterName: 'CVE Type',
                },
            ]);
        });
    });
});
