import React from 'react';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';

import PageNotFound from 'Components/PageNotFound';
import VulnMgmtEntityDeployment from './Deployment/VulnMgmtEntityDeployment';
import VulnMgmtEntityImage from './Image/VulnMgmtEntityImage';
import VulnMgmtEntityNodeComponent from './Component/VulnMgmtEntityNodeComponent';
import VulnMgmtEntityImageComponent from './Component/VulnMgmtEntityImageComponent';
import VulnMgmtEntityCve from './Cve/VulnMgmtEntityCve';
import VulnMgmtEntityCluster from './Cluster/VulnMgmtEntityCluster';
import VulnMgmtEntityNamespace from './Namespace/VulnMgmtEntityNamespace';
import VulnMgmtEntityNode from './Node/VulnMgmtEntityNode';

const entityComponentMap = {
    [entityTypes.DEPLOYMENT]: VulnMgmtEntityDeployment,
    [entityTypes.IMAGE]: VulnMgmtEntityImage,
    [entityTypes.NODE_COMPONENT]: VulnMgmtEntityNodeComponent,
    [entityTypes.IMAGE_COMPONENT]: VulnMgmtEntityImageComponent,
    [entityTypes.IMAGE_CVE]: VulnMgmtEntityCve,
    [entityTypes.NODE_CVE]: VulnMgmtEntityCve,
    [entityTypes.CLUSTER_CVE]: VulnMgmtEntityCve,
    [entityTypes.CLUSTER]: VulnMgmtEntityCluster,
    [entityTypes.NAMESPACE]: VulnMgmtEntityNamespace,
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
