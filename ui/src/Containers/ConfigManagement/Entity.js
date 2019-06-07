import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';

import PageNotFound from 'Components/PageNotFound';
import ServiceAccount from './Entity/ServiceAccount';

const entityComponentMap = {
    [entityTypes.SERVICE_ACCOUNT]: ServiceAccount
};

const Entity = ({ entityType, entityId }) => {
    const Component = entityComponentMap[entityType];
    if (!Component) return <PageNotFound resourceType={entityType} />;
    return <Component id={entityId} />;
};

Entity.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired
};

export default Entity;
