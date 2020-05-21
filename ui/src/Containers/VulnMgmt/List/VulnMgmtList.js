import React from 'react';
import entityTypes from 'constants/entityTypes';

import PageNotFound from 'Components/PageNotFound';
import VulnMgmtListDeployments from './Deployments/VulnMgmtListDeployments';
import VulnMgmtListImages from './Images/VulnMgmtListImages';
import VulnMgmtListComponents from './Components/VulnMgmtListComponents';
import VulnMgmtListCves from './Cves/VulnMgmtListCves';
import VulnMgmtListClusters from './Clusters/VulnMgmtListClusters';
import VulnMgmtListNamespaces from './Namespaces/VulnMgmtListNamespaces';
import VulnMgmtListPolicies from './Policies/VulnMgmtListPolicies';

const entityComponentMap = {
    [entityTypes.DEPLOYMENT]: VulnMgmtListDeployments,
    [entityTypes.IMAGE]: VulnMgmtListImages,
    [entityTypes.COMPONENT]: VulnMgmtListComponents,
    [entityTypes.CVE]: VulnMgmtListCves,
    [entityTypes.CLUSTER]: VulnMgmtListClusters,
    [entityTypes.NAMESPACE]: VulnMgmtListNamespaces,
    [entityTypes.POLICY]: VulnMgmtListPolicies,
};

const VulnMgmtEntityList = (props) => {
    const { entityListType } = props;
    const Component = entityComponentMap[entityListType];
    if (!Component) return <PageNotFound resourceType={entityListType} />;
    return <Component {...props} />;
};

export default VulnMgmtEntityList;
