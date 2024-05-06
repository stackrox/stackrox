import {
    deploymentSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageSearchFilterConfig,
} from '../types';
import { getEntities, getEntityAttributes, getDefaultEntity, getDefaultAttribute } from './utils';

describe('utils', () => {
    describe('getEntities', () => {
        it('should get the entities in a config object', () => {
            const config = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                ImageCVE: imageCVESearchFilterConfig,
            };

            const result = getEntities(config);

            expect(result).toStrictEqual(['Image', 'Deployment', 'ImageCVE']);
        });
    });

    describe('getEntityAttributes', () => {
        it('should get the attributes of an entity in a config object', () => {
            const config = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                ImageCVE: imageCVESearchFilterConfig,
            };

            const result = getEntityAttributes('ImageCVE', config);

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
                    inputType: 'dropdown-slider',
                },
                {
                    displayName: 'Type',
                    filterChipLabel: 'Image CVE Type',
                    searchTerm: 'CVE Type',
                    inputType: 'select',
                },
            ]);
        });
    });

    describe('getDefaultEntity', () => {
        it('should get the default (first) entity in a config object', () => {
            const config = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                ImageCVE: imageCVESearchFilterConfig,
            };

            const result = getDefaultEntity(config);

            expect(result).toStrictEqual('Image');
        });
    });

    describe('getDefaultAttribute', () => {
        it('should get the default (first) attribute of a specific entity in a config object', () => {
            const config = {
                Image: imageSearchFilterConfig,
                Deployment: deploymentSearchFilterConfig,
                ImageCVE: imageCVESearchFilterConfig,
            };

            const result = getDefaultAttribute('ImageCVE', config);

            expect(result).toStrictEqual('ID');
        });
    });
});
