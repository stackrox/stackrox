import React from 'react';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';

import PageNotFound from 'Components/PageNotFound';
import VulnMgmtEntityDeployment from './Deployment/VulnMgmtEntityDeployment';
import VulnMgmtEntityImage from './Image/VulnMgmtEntityImage';
import VulnMgmtEntityComponent from './Component/VulnMgmtEntityComponent';
import VulnMgmtEntityCve from './Cve/VulnMgmtEntityCve';
import VulnMgmtEntityCluster from './Cluster/VulnMgmtEntityCluster';
import VulnMgmtEntityNamespace from './Namespace/VulnMgmtEntityNamespace';
import VulnMgmtEntityPolicy from './Policy/VulnMgmtEntityPolicy';
import VulnMgmtEntityNode from './Node/VulnMgmtEntityNode';

const entityComponentMap = {
    [entityTypes.DEPLOYMENT]: VulnMgmtEntityDeployment,
    [entityTypes.IMAGE]: VulnMgmtEntityImage,
    [entityTypes.COMPONENT]: VulnMgmtEntityComponent,
    [entityTypes.CVE]: VulnMgmtEntityCve,
    [entityTypes.IMAGE_CVE]: VulnMgmtEntityCve,
    [entityTypes.NODE_CVE]: VulnMgmtEntityCve,
    [entityTypes.CLUSTER_CVE]: VulnMgmtEntityCve,
    [entityTypes.CLUSTER]: VulnMgmtEntityCluster,
    [entityTypes.NAMESPACE]: VulnMgmtEntityNamespace,
    [entityTypes.POLICY]: VulnMgmtEntityPolicy,
    [entityTypes.NODE]: VulnMgmtEntityNode,
};

const VulnMgmtEntity = (props) => {
    const { entityType } = props;
    const Component = entityComponentMap[entityType];
    if (!Component) {
        return <PageNotFound resourceType={entityType} useCase={useCases.VULN_MANAGEMENT} />;
    }
    return <Component {...props} />;
};

export default VulnMgmtEntity;
