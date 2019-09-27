import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';

import PageNotFound from 'Components/PageNotFound';
import VulnMgmtDeployments from './VulnMgmtDeployments';
import VulnMgmtImages from './VulnMgmtImages';
import VulnMgmtComponents from './VulnMgmtComponents';
import VulnMgmtCves from './VulnMgmtCves';
import VulnMgmtClusters from './VulnMgmtClusters';
import VulnMgmtNamespaces from './VulnMgmtNamespaces';
import VulnMgmtPolicies from './VulnMgmtPolicies';

const entityComponentMap = {
    [entityTypes.DEPLOYMENT]: VulnMgmtDeployments,
    [entityTypes.IMAGE]: VulnMgmtImages,
    [entityTypes.COMPONENT]: VulnMgmtComponents,
    [entityTypes.CVE]: VulnMgmtCves,
    [entityTypes.CLUSTER]: VulnMgmtClusters,
    [entityTypes.NAMESPACE]: VulnMgmtNamespaces,
    [entityTypes.POLICY]: VulnMgmtPolicies
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
