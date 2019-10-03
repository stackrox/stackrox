import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';

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
    [entityTypes.POLICY]: VulnMgmtListPolicies
};

const VulnMgmtEntityList = ({ entityListType, entityId, ...rest }) => {
    const Component = entityComponentMap[entityListType];
    if (!Component) return <PageNotFound resourceType={entityListType} />;
    return <Component selectedRowId={entityId} {...rest} />;
};

VulnMgmtEntityList.propTypes = {
    entityListType: PropTypes.string.isRequired,
    entityId: PropTypes.string
};

VulnMgmtEntityList.defaultProps = {
    entityId: null
};

export default withRouter(VulnMgmtEntityList);
