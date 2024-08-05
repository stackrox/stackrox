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

const nodeSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Node',
    searchCategory: 'NODES',
    attributes: nodeAttributes,
};

const nodeCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'NODE_VULNERABILITIES',
    attributes: nodeCVEAttributes,
};

const nodeComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS',
    attributes: nodeComponentAttributes,
};

const imageSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image',
    searchCategory: 'IMAGES',
    attributes: imageAttributes,
};

const imageCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES',
    attributes: imageCVEAttributes,
};

const imageComponentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS',
    attributes: imageComponentAttributes,
};

const deploymentSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS',
    attributes: deploymentAttributes,
};

const namespaceSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES',
    attributes: namespaceAttributes,
};

const clusterSearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS',
    attributes: clusterAttributes,
};

const platformCVESearchFilterConfig: CompoundSearchFilterEntity = {
    displayName: 'CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES',
    attributes: platformCVEAttributes,
};

export {
    nodeSearchFilterConfig,
    nodeCVESearchFilterConfig,
    nodeComponentSearchFilterConfig,
    imageSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
    platformCVESearchFilterConfig,
};
