import { getImageAttributes } from '../attributes/image';
import { createSearchFilterConfig } from './searchFilterConfig';

describe('searchFilterConfig', () => {
    describe('createSearchFilterConfig', () => {
        it('should get the image config with selected attributes', () => {
            const result = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(['Name', 'Tag', 'Label']),
                },
            ]);

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
        it('should get the image config with all attributes if none were selected', () => {
            const result = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(),
                },
            ]);

            expect(result).toStrictEqual({
                displayName: 'Image',
                searchCategory: 'IMAGES',
                attributes: {},
            });
        });
    });
});
