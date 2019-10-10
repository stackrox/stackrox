import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';

import PageNotFound from 'Components/PageNotFound';
import VulnMgmtEntityDeployment from './Deployment/VulnMgmtEntityDeployment';
import VulnMgmtEntityImage from './Image/VulnMgmtEntityImage';
import VulnMgmtEntityComponent from './Component/VulnMgmtEntityComponent';
import VulnMgmtEntityCve from './Cve/VulnMgmtEntityCve';
import VulnMgmtEntityCluster from './Cluster/VulnMgmtEntityCluster';
import VulnMgmtEntityNamespace from './Namespace/VulnMgmtEntityNamespace';
import VulnMgmtEntityPolicy from './Policy/VulnMgmtEntityPolicy';

const entityComponentMap = {
    [entityTypes.DEPLOYMENT]: VulnMgmtEntityDeployment,
    [entityTypes.IMAGE]: VulnMgmtEntityImage,
    [entityTypes.COMPONENT]: VulnMgmtEntityComponent,
    [entityTypes.CVE]: VulnMgmtEntityCve,
    [entityTypes.CLUSTER]: VulnMgmtEntityCluster,
    [entityTypes.NAMESPACE]: VulnMgmtEntityNamespace,
    [entityTypes.POLICY]: VulnMgmtEntityPolicy
};

const VulnMgmtEntity = ({ entityType, entityId, entityListType, search, entityContext }) => {
    const Component = entityComponentMap[entityType];
    if (!Component) return <PageNotFound resourceType={entityType} />;
    return (
        <Component
            entityId={entityId}
            entityListType={entityListType}
            search={search}
            entityContext={entityContext}
        />
    );
};

VulnMgmtEntity.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    search: PropTypes.shape({}),
    entityContext: PropTypes.shape({})
};

VulnMgmtEntity.defaultProps = {
    entityListType: null,
    search: null,
    entityContext: {}
};

export default VulnMgmtEntity;
