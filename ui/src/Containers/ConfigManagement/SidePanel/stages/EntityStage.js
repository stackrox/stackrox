import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import URLService from 'modules/URLService';

import EntityOverview from 'Containers/ConfigManagement/Entity';

const EntityStage = ({ match, location, history, entityType1, entityId1 }) => {
    function onRelatedEntityClick(entityType, entityId) {
        const urlBuilder = URLService.getURL(match, location).push(entityType, entityId);
        history.push(urlBuilder.url());
    }

    function onRelatedEntityListClick(entityListType) {
        const urlBuilder = URLService.getURL(match, location).push(entityListType);
        history.push(urlBuilder.url());
    }

    return (
        <EntityOverview
            entityType={entityType1}
            entityId={entityId1}
            onRelatedEntityClick={onRelatedEntityClick}
            onRelatedEntityListClick={onRelatedEntityListClick}
        />
    );
};

EntityStage.propTypes = {
    match: ReactRouterPropTypes.match,
    location: ReactRouterPropTypes.location,
    history: ReactRouterPropTypes.history,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string
};

EntityStage.defaultProps = {
    match: null,
    location: null,
    history: null,
    entityType1: null,
    entityId1: null
};

export default EntityStage;
