import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';

import PageNotFound from 'Components/PageNotFound';
import ServiceAccount from './Entity/ServiceAccount';
import Secret from './Entity/Secret';
import Deployment from './Entity/Deployment';
import Cluster from './Entity/Cluster';
import Namespace from './Entity/Namespace';

const entityComponentMap = {
    [entityTypes.SERVICE_ACCOUNT]: ServiceAccount,
    [entityTypes.SECRET]: Secret,
    [entityTypes.DEPLOYMENT]: Deployment,
    [entityTypes.CLUSTER]: Cluster,
    [entityTypes.NAMESPACE]: Namespace
};

const Entity = ({ entityType, entityId, onRelatedEntityClick, onRelatedEntityListClick }) => {
    const Component = entityComponentMap[entityType];
    if (!Component) return <PageNotFound resourceType={entityType} />;
    return (
        <Component
            id={entityId}
            onRelatedEntityClick={onRelatedEntityClick}
            onRelatedEntityListClick={onRelatedEntityListClick}
        />
    );
};

Entity.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    onRelatedEntityClick: PropTypes.func.isRequired,
    onRelatedEntityListClick: PropTypes.func.isRequired
};

export default Entity;
