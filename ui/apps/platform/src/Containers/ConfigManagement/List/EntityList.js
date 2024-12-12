import React from 'react';
import PropTypes from 'prop-types';

import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';

import PageNotFound from 'Components/PageNotFound';
import Namespaces from './Namespaces';
import Subjects from './Subjects';
import ServiceAccounts from './ServiceAccounts';
import Clusters from './Clusters';
import Nodes from './Nodes';
import Deployments from './Deployments';
import Secrets from './Secrets';
import Roles from './Roles';
import Images from './Images';
import Policies from './Policies';
import CISControls from './CISControls';

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
    [entityTypes.CONTROL]: CISControls,
};

const EntityList = ({ entityListType, entityId, ...rest }) => {
    const Component = entityComponentMap[entityListType];
    if (!Component) {
        return <PageNotFound resourceType={entityListType} useCase={useCases.CONFIG_MANAGEMENT} />;
    }
    return <Component selectedRowId={entityId} {...rest} />;
};

EntityList.propTypes = {
    entityListType: PropTypes.string.isRequired,
    entityId: PropTypes.string,
};

EntityList.defaultProps = {
    entityId: null,
};

export default EntityList;
