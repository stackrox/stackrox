import { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';
import { Name as ImageName } from 'Components/CompoundSearchFilter/attributes/image';
import { Name as ImageCveName } from 'Components/CompoundSearchFilter/attributes/imageCVE';
import { vulnerabilityRequestAttributes } from 'Components/CompoundSearchFilter/attributes/vulnerabilityRequests';

const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: [ImageName],
};

const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES_V2', // flat CVE data model
    attributes: [ImageCveName],
};

const vulnerabilityRequestSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Exception',
    searchCategory: 'VULN_REQUEST',
    attributes: vulnerabilityRequestAttributes,
};

export const vulnRequestSearchFilterConfig = [
    vulnerabilityRequestSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageSearchFilterConfig,
];
