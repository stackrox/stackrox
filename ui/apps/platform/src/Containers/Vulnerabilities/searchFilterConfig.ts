import { SearchCategory } from 'services/SearchService';
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

const nodeSearchFilterConfig = {
    displayName: 'Node',
    searchCategory: 'NODES' as SearchCategory,
    attributes: nodeAttributes,
};

const nodeCVESearchFilterConfig = {
    displayName: 'CVE',
    searchCategory: 'NODE_VULNERABILITIES' as SearchCategory,
    attributes: nodeCVEAttributes,
};

const nodeComponentSearchFilterConfig = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS' as SearchCategory,
    attributes: nodeComponentAttributes,
};

const imageSearchFilterConfig = {
    displayName: 'Image',
    searchCategory: 'IMAGES' as SearchCategory,
    attributes: imageAttributes,
};

const imageCVESearchFilterConfig = {
    displayName: 'CVE',
    searchCategory: 'IMAGE_VULNERABILITIES' as SearchCategory,
    attributes: imageCVEAttributes,
};

const imageComponentSearchFilterConfig = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS' as SearchCategory,
    attributes: imageComponentAttributes,
};

const deploymentSearchFilterConfig = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS' as SearchCategory,
    attributes: deploymentAttributes,
};

const namespaceSearchFilterConfig = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES' as SearchCategory,
    attributes: namespaceAttributes,
};

const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS' as SearchCategory,
    attributes: clusterAttributes,
};

const platformCVESearchFilterConfig = {
    displayName: 'CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES' as SearchCategory,
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
