import entityTypes from 'constants/entityTypes';

import { defaultClusterSort } from './List/Clusters/VulnMgmtListClusters';
import { defaultComponentSort } from './List/Components/VulnMgmtListComponents';
import { defaultCveSort } from './List/Cves/VulnMgmtListCves';
import { defaultDeploymentSort } from './List/Deployments/VulnMgmtListDeployments';
import { defaultImageSort } from './List/Images/VulnMgmtListImages';
import { defaultNamespaceSort } from './List/Namespaces/VulnMgmtListNamespaces';
import { defaultNodeSort } from './List/Nodes/VulnMgmtListNodes';

const vulnMgmtDefaultSorts = {
    [entityTypes.CLUSTER]: defaultClusterSort,
    [entityTypes.COMPONENT]: defaultComponentSort,
    [entityTypes.NODE_COMPONENT]: defaultComponentSort,
    [entityTypes.IMAGE_COMPONENT]: defaultComponentSort,
    [entityTypes.CVE]: defaultCveSort,
    [entityTypes.IMAGE_CVE]: defaultCveSort,
    [entityTypes.NODE_CVE]: defaultCveSort,
    [entityTypes.CLUSTER_CVE]: defaultCveSort,
    [entityTypes.DEPLOYMENT]: defaultDeploymentSort,
    [entityTypes.IMAGE]: defaultImageSort,
    [entityTypes.NAMESPACE]: defaultNamespaceSort,
    [entityTypes.NODE]: defaultNodeSort,
};

export default vulnMgmtDefaultSorts;
