import { getDeploymentAttributes } from '../attributes/deployment';
import { getImageAttributes } from '../attributes/image';
import { getImageCVEAttributes } from '../attributes/imageCVE';
import { createSearchFilterConfig } from './searchFilterConfig';
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
            const config = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(),
                },
                {
                    displayName: 'Deployment',
                    searchCategory: 'DEPLOYMENTS',
                    attributes: getDeploymentAttributes(),
                },
                {
                    displayName: 'Image CVE',
                    searchCategory: 'IMAGE_VULNERABILITIES',
                    attributes: getImageCVEAttributes(),
                },
            ]);

            const result = getEntities(config);

            expect(result).toStrictEqual(['Image', 'Deployment', 'Image CVE']);
        });
    });

    describe('getEntityAttributes', () => {
        it('should get the attributes of an entity in a config object', () => {
            const config = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(),
                },
                {
                    displayName: 'Deployment',
                    searchCategory: 'DEPLOYMENTS',
                    attributes: getDeploymentAttributes(),
                },
                {
                    displayName: 'Image CVE',
                    searchCategory: 'IMAGE_VULNERABILITIES',
                    attributes: getImageCVEAttributes(),
                },
            ]);

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
            const config = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(),
                },
                {
                    displayName: 'Deployment',
                    searchCategory: 'DEPLOYMENTS',
                    attributes: getDeploymentAttributes(),
                },
                {
                    displayName: 'Image CVE',
                    searchCategory: 'IMAGE_VULNERABILITIES',
                    attributes: getImageCVEAttributes(),
                },
            ]);

            const result = getDefaultEntity(config);

            expect(result).toStrictEqual('Image');
        });
    });

    describe('getDefaultAttribute', () => {
        it('should get the default (first) attribute of a specific entity in a config object', () => {
            const config = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(),
                },
                {
                    displayName: 'Deployment',
                    searchCategory: 'DEPLOYMENTS',
                    attributes: getDeploymentAttributes(),
                },
                {
                    displayName: 'Image CVE',
                    searchCategory: 'IMAGE_VULNERABILITIES',
                    attributes: getImageCVEAttributes(),
                },
            ]);

            const result = getDefaultAttribute('Image CVE', config);

            expect(result).toStrictEqual('Name');
        });
    });

    describe('makeFilterChipDescriptors', () => {
        it('should create an array of FilterChipGroupDescriptor objects from a config object', () => {
            const config = createSearchFilterConfig([
                {
                    displayName: 'Image CVE',
                    searchCategory: 'IMAGE_VULNERABILITIES',
                    attributes: getImageCVEAttributes(),
                },
            ]);

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
