import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';

import ServiceAccount from './ServiceAccount';

const entityComponentMap = {
    [entityTypes.SERVICE_ACCOUNT]: ServiceAccount
};

const EntityOverview = ({ entityType, entityId }) => {
    const Component = entityComponentMap[entityType];
    return <Component id={entityId} />;
};

EntityOverview.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired
};

export default EntityOverview;
