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
                Image: {
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
                },
            });
        });

        it('should get the image config with all attributes if none were selected', () => {
            const result = createSearchFilterConfig([
                {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: getImageAttributes(),
                },
            ]);

            expect(result).toStrictEqual({
                Image: {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: {
                        Name: {
                            displayName: 'Name',
                            filterChipLabel: 'Image name',
                            searchTerm: 'Image',
                            inputType: 'autocomplete',
                        },
                        'Operating system': {
                            displayName: 'Operating system',
                            filterChipLabel: 'Image operating system',
                            searchTerm: 'Image OS',
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
                        Registry: {
                            displayName: 'Registry',
                            filterChipLabel: 'Image registry',
                            searchTerm: 'Image Registry',
                            inputType: 'text',
                        },
                    },
                },
            });
        });
    });
});
