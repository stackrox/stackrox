import { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';
import { Name as ImageName } from 'Components/CompoundSearchFilter/attributes/image';
import { Name as ImageCveName } from 'Components/CompoundSearchFilter/attributes/imageCVE';
import { vulnerabilityRequestAttributes } from 'Components/CompoundSearchFilter/attributes/vulnerabilityRequests';

const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: [ImageName],
};

// After release, delete.
const imageCVESearchFilterConfigV1: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES',
    attributes: [ImageCveName],
};

const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES_V2',
    attributes: [ImageCveName],
};

const vulnerabilityRequestSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Exception',
    searchCategory: 'VULN_REQUEST',
    attributes: vulnerabilityRequestAttributes,
};

// After release, delete.
const vulnRequestSearchFilterConfigV1 = [
    vulnerabilityRequestSearchFilterConfig,
    imageCVESearchFilterConfigV1,
    imageSearchFilterConfig,
];

const vulnRequestSearchFilterConfig = [
    vulnerabilityRequestSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageSearchFilterConfig,
];

// After release, replace temporary function
// with export of vulnRequestSearchFilterConfig (above)
// that has unconditional updated imageCVESearchFilterConfig.
export function convertToFlatVulnRequestSearchFilterConfig(
    isFlattenCveDataEnabled: boolean // ROX_FLATTEN_CVE_DATA
): CompoundSearchFilterEntity[] {
    return isFlattenCveDataEnabled
        ? vulnRequestSearchFilterConfig
        : vulnRequestSearchFilterConfigV1;
}
