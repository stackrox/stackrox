import { CompoundSearchFilterConfig, DeepRequired } from '../types';

// We want to only retrieve the "Image" config and make sure all the "attributes" are required (not optional)
export type ImageCompoundSearchFilterConfig = DeepRequired<
    Required<Pick<CompoundSearchFilterConfig, 'Image'>>
>;

export const imageFilterConfig: ImageCompoundSearchFilterConfig = {
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
            OperatingSystem: {
                displayName: 'Operating System',
                filterChipLabel: 'Image Operating System',
                searchTerm: 'Image OS',
                inputType: 'text',
            },
            Tag: {
                displayName: 'Tag',
                filterChipLabel: 'Image Tag',
                searchTerm: 'Image Tag',
                inputType: 'text',
            },
            CVSS: {
                displayName: 'CVSS',
                filterChipLabel: 'Image CVSS',
                searchTerm: 'Image Top CVSS',
                inputType: 'dropdown-slider',
            },
            Label: {
                displayName: 'Label',
                filterChipLabel: 'Image Label',
                searchTerm: 'Image Label',
                inputType: 'autocomplete',
            },
            CreatedTime: {
                displayName: 'Created Time',
                filterChipLabel: 'Image Created Time',
                searchTerm: 'Image Created Time',
                inputType: 'date-picker',
            },
            ScanTime: {
                displayName: 'Scan Time',
                filterChipLabel: 'Image Scan Time',
                searchTerm: 'Image Scan Time',
                inputType: 'date-picker',
            },
            Registry: {
                displayName: 'Registry',
                filterChipLabel: 'Image Registry',
                searchTerm: 'Image Registry',
                inputType: 'text',
            },
        },
    },
};

export type ImageAttribute =
    | 'Name'
    | 'OperatingSystem'
    | 'Tag'
    | 'CVSS'
    | 'Label'
    | 'CreatedTime'
    | 'ScanTime'
    | 'Registry';

export function getImageFilterConfig(selectedAttributes: ImageAttribute[]) {
    const filteredAttributes = {};

    selectedAttributes.forEach((attributeKey) => {
        filteredAttributes[attributeKey] = imageFilterConfig.Image.attributes[attributeKey];
    });

    const modifiedImageFilterConfig: CompoundSearchFilterConfig = {
        Image: {
            displayName: 'Image',
            searchCategory: 'IMAGES',
            attributes: filteredAttributes,
        },
    };

    return modifiedImageFilterConfig;
}
