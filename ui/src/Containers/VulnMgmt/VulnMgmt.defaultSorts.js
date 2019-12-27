import entityTypes from 'constants/entityTypes';

import { defaultClusterSort } from './List/Clusters/VulnMgmtListClusters';
import { defaultComponentSort } from './List/Components/VulnMgmtListComponents';
import { defaultCveSort } from './List/Cves/VulnMgmtListCves';
import { defaultDeploymentSort } from './List/Deployments/VulnMgmtListDeployments';
import { defaultImageSort } from './List/Images/VulnMgmtListImages';
import { defaultNamespaceSort } from './List/Namespaces/VulnMgmtListNamespaces';
import { defaultPolicySort } from './List/Policies/VulnMgmtListPolicies';

const vulnMgmtDefaultSorts = {
    [entityTypes.CLUSTER]: defaultClusterSort,
    [entityTypes.COMPONENT]: defaultComponentSort,
    [entityTypes.CVE]: defaultCveSort,
    [entityTypes.DEPLOYMENT]: defaultDeploymentSort,
    [entityTypes.IMAGE]: defaultImageSort,
    [entityTypes.NAMESPACE]: defaultNamespaceSort,
    [entityTypes.POLICY]: defaultPolicySort
};

export default vulnMgmtDefaultSorts;
