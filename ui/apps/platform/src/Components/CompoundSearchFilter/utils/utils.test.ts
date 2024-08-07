import { deploymentAttributes } from '../attributes/deployment';
import { imageAttributes } from '../attributes/image';
import { imageCVEAttributes } from '../attributes/imageCVE';
import {
    getDefaultAttributeName,
    getDefaultEntityName,
    getEntityAttributes,
    makeFilterChipDescriptors,
} from './utils';
import { CompoundSearchFilterEntity } from '../types';

const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: imageAttributes,
};

const deploymentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: deploymentAttributes,
};

const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image CVE',
    searchCategory: 'IMAGE_VULNERABILITIES',
    attributes: imageCVEAttributes,
};

describe('utils', () => {
    describe('getEntities', () => {
        it('should get the entities in a config object', () => {
            const config = [
                imageSearchFilterConfig,
                deploymentSearchFilterConfig,
                imageCVESearchFilterConfig,
            ];

            expect(config).toStrictEqual([
                imageSearchFilterConfig,
                deploymentSearchFilterConfig,
                imageCVESearchFilterConfig,
            ]);
        });
    });

    describe('getEntityAttributes', () => {
        it('should get the attributes of an entity in a config object', () => {
            const config = [
                imageSearchFilterConfig,
                deploymentSearchFilterConfig,
                imageCVESearchFilterConfig,
            ];

            const result = getEntityAttributes(config, 'Image CVE');

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
            const config = [
                imageSearchFilterConfig,
                deploymentSearchFilterConfig,
                imageCVESearchFilterConfig,
            ];

            const result = getDefaultEntityName(config);

            expect(result).toStrictEqual('Image');
        });
    });

    describe('getDefaultAttribute', () => {
        it('should get the default (first) attribute of a specific entity in a config object', () => {
            const config = [
                imageSearchFilterConfig,
                deploymentSearchFilterConfig,
                imageCVESearchFilterConfig,
            ];

            const result = getDefaultAttributeName(config, 'Image CVE');

            expect(result).toStrictEqual('Name');
        });
    });

    describe('makeFilterChipDescriptors', () => {
        it('should create an array of FilterChipGroupDescriptor objects from a config object', () => {
            const config = [imageCVESearchFilterConfig];

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
