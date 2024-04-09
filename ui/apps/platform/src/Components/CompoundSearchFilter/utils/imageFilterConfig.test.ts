import { getImageFilterConfig } from './imageFilterConfig';

describe('imageFilterConfig', () => {
    describe('getImageFilterConfig', () => {
        it('should get the image config with selected attributes', () => {
            const result = getImageFilterConfig(['Name', 'Label', 'Tag']);

            expect(result).toStrictEqual({
                Image: {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: {
                        Name: {
                            displayName: 'Name',
                            filterChipLabel: 'Image Name',
                            searchTerm: 'Image',
                            inputType: 'autocomplete',
                        },
                        Tag: {
                            displayName: 'Tag',
                            filterChipLabel: 'Image Tag',
                            searchTerm: 'Image Tag',
                            inputType: 'text',
                        },
                        Label: {
                            displayName: 'Label',
                            filterChipLabel: 'Image Label',
                            searchTerm: 'Image Label',
                            inputType: 'autocomplete',
                        },
                    },
                },
            });
        });

        // We will allow this case. If the Compound Search Filter gets a config like this, it'll just ignore it.
        it('should get the image config with no attributes if none were selected', () => {
            const result = getImageFilterConfig([]);

            expect(result).toStrictEqual({
                Image: {
                    displayName: 'Image',
                    searchCategory: 'IMAGES',
                    attributes: {},
                },
            });
        });
    });
});
