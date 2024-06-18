import { imageSearchFilterConfig } from '../types';
import { getFilteredConfig } from './searchFilterConfig';

describe('searchFilterConfig', () => {
    describe('getFilteredConfig', () => {
        it('should get the image config with selected attributes', () => {
            const result = getFilteredConfig(imageSearchFilterConfig, ['Name', 'Tag', 'Label']);

            expect(result).toStrictEqual({
                displayName: 'Image',
                searchCategory: 'IMAGES',
                attributes: {
                    Name: {
                        displayName: 'Name',
                        filterChipLabel: 'Image name',
                        searchTerm: 'Image',
                        inputType: 'autocomplete',
                    },
                    Tag: {
                        displayName: 'Tag',
                        filterChipLabel: 'Image tag',
                        searchTerm: 'Image Tag',
                        inputType: 'text',
                    },
                    Label: {
                        displayName: 'Label',
                        filterChipLabel: 'Image label',
                        searchTerm: 'Image Label',
                        inputType: 'autocomplete',
                    },
                },
            });
        });

        // We will allow this case. If the Compound Search Filter gets a config like this, it'll just ignore it.
        it('should get the image config with no attributes if none were selected', () => {
            const result = getFilteredConfig(imageSearchFilterConfig, []);

            expect(result).toStrictEqual({
                displayName: 'Image',
                searchCategory: 'IMAGES',
                attributes: {},
            });
        });
    });
});
