import { SearchCategory } from 'services/SearchService';
import { getClusterAttributes } from 'Components/CompoundSearchFilter/attributes/cluster';
import { getDeploymentAttributes } from 'Components/CompoundSearchFilter/attributes/deployment';
import { getImageAttributes } from 'Components/CompoundSearchFilter/attributes/image';
import { getImageCVEAttributes } from 'Components/CompoundSearchFilter/attributes/imageCVE';
import { getImageComponentAttributes } from 'Components/CompoundSearchFilter/attributes/imageComponent';
import { getNamespaceAttributes } from 'Components/CompoundSearchFilter/attributes/namespace';
import { getNodeAttributes } from 'Components/CompoundSearchFilter/attributes/node';
import { getNodeComponentAttributes } from 'Components/CompoundSearchFilter/attributes/nodeComponent';
import { getPlatformCVEAttributes } from 'Components/CompoundSearchFilter/attributes/platformCVE';

const nodeSearchFilterConfig = {
    displayName: 'Node',
    searchCategory: 'NODES' as SearchCategory,
    attributes: getNodeAttributes(),
};

const nodeComponentSearchFilterConfig = {
    displayName: 'Node component',
    searchCategory: 'NODE_COMPONENTS' as SearchCategory,
    attributes: getNodeComponentAttributes(),
};

const imageSearchFilterConfig = {
    displayName: 'Image',
    searchCategory: 'IMAGES' as SearchCategory,
    attributes: getImageAttributes(),
};

const imageCVESearchFilterConfig = {
    displayName: 'Image CVE',
    searchCategory: 'IMAGES_VULNERABILITIES' as SearchCategory,
    attributes: getImageCVEAttributes(),
};

const imageComponentSearchFilterConfig = {
    displayName: 'Image component',
    searchCategory: 'IMAGE_COMPONENTS' as SearchCategory,
    attributes: getImageComponentAttributes(),
};

const deploymentSearchFilterConfig = {
    displayName: 'Deployment',
    searchCategory: 'DEPLOYMENTS' as SearchCategory,
    attributes: getDeploymentAttributes(),
};

const namespaceSearchFilterConfig = {
    displayName: 'Namespace',
    searchCategory: 'NAMESPACES' as SearchCategory,
    attributes: getNamespaceAttributes(),
};

const clusterSearchFilterConfig = {
    displayName: 'Cluster',
    searchCategory: 'CLUSTERS' as SearchCategory,
    attributes: getClusterAttributes(),
};

const platformCVESearchFilterConfig = {
    displayName: 'Platform CVE',
    searchCategory: 'CLUSTER_VULNERABILITIES' as SearchCategory,
    attributes: getPlatformCVEAttributes(),
};

export {
    nodeSearchFilterConfig,
    nodeComponentSearchFilterConfig,
    imageSearchFilterConfig,
    imageCVESearchFilterConfig,
    imageComponentSearchFilterConfig,
    deploymentSearchFilterConfig,
    namespaceSearchFilterConfig,
    clusterSearchFilterConfig,
    platformCVESearchFilterConfig,
};
