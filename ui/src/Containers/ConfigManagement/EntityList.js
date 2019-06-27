import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { withRouter } from 'react-router-dom';

import PageNotFound from 'Components/PageNotFound';
import Namespaces from './List/Namespaces';
import Subjects from './List/Subjects';
import ServiceAccounts from './List/ServiceAccounts';
import Clusters from './List/Clusters';
import Nodes from './List/Nodes';
import Deployments from './List/Deployments';
import Secrets from './List/Secrets';
import Roles from './List/Roles';
import Images from './List/Images';
import Policies from './List/Policies';
import CISControls from './List/CISControls';

const entityComponentMap = {
    [entityTypes.SUBJECT]: Subjects,
    [entityTypes.SERVICE_ACCOUNT]: ServiceAccounts,
    [entityTypes.CLUSTER]: Clusters,
    [entityTypes.NAMESPACE]: Namespaces,
    [entityTypes.NODE]: Nodes,
    [entityTypes.DEPLOYMENT]: Deployments,
    [entityTypes.IMAGE]: Images,
    [entityTypes.SECRET]: Secrets,
    [entityTypes.ROLE]: Roles,
    [entityTypes.POLICY]: Policies,
    [entityTypes.CONTROL]: CISControls
};

const EntityList = ({ entityListType, entityId, ...rest }) => {
    const Component = entityComponentMap[entityListType];
    if (!Component) return <PageNotFound resourceType={entityListType} />;
    return <Component selectedRowId={entityId} {...rest} />;
};

EntityList.propTypes = {
    entityListType: PropTypes.string.isRequired,
    entityId: PropTypes.string
};

EntityList.defaultProps = {
    entityId: null
};

export default withRouter(EntityList);
