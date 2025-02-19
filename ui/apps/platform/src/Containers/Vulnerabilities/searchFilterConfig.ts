import { CompoundSearchFilterEntity } from 'Components/CompoundSearchFilter/types';
import { clusterAttributes } from 'Components/CompoundSearchFilter/attributes/cluster';
import { deploymentAttributes } from 'Components/CompoundSearchFilter/attributes/deployment';
import { imageAttributes } from 'Components/CompoundSearchFilter/attributes/image';
import { imageCVEAttributes } from 'Components/CompoundSearchFilter/attributes/imageCVE';
import { imageComponentAttributes } from 'Components/CompoundSearchFilter/attributes/imageComponent';
import { namespaceAttributes } from 'Components/CompoundSearchFilter/attributes/namespace';
import { nodeAttributes } from 'Components/CompoundSearchFilter/attributes/node';
import { nodeCVEAttributes } from 'Components/CompoundSearchFilter/attributes/nodeCVE';
import { nodeComponentAttributes } from 'Components/CompoundSearchFilter/attributes/nodeComponent';
import { platformCVEAttributes } from 'Components/CompoundSearchFilter/attributes/platformCVE';

export const nodeSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: nodeAttributes,
};

export const nodeCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'NODE_VULNERABILITIES',
    attributes: nodeCVEAttributes,
};

export const nodeComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS',
    attributes: nodeComponentAttributes,
};

export const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: imageAttributes,
};

export const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES',
    attributes: imageCVEAttributes,
};

export const imageComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS',
    attributes: imageComponentAttributes,
};

export const deploymentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: deploymentAttributes,
};

export const namespaceSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: namespaceAttributes,
};

export const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: clusterAttributes,
};

export const platformCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES',
    attributes: platformCVEAttributes,
};
