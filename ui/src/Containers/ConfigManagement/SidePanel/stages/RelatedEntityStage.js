import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import URLService from 'modules/URLService';

import EntityOverview from 'Containers/ConfigManagement/Entity';

const RelatedEntityStage = ({
    match,
    location,
    history,
    entityType2,
    entityListType2,
    entityId2
}) => {
    const relatedEntityType = entityType2 || entityListType2;

    function onRelatedEntityClick(entityType, entityId) {
        const urlBuilder = URLService.getURL(match, location).base(entityType, entityId);
        history.push(urlBuilder.url());
    }

    function onRelatedEntityListClick(entityListType) {
        const urlBuilder = URLService.getURL(match, location).base(entityListType);
        history.push(urlBuilder.url());
    }

    return (
        <EntityOverview
            entityType={relatedEntityType}
            entityId={entityId2}
            onRelatedEntityClick={onRelatedEntityClick}
            onRelatedEntityListClick={onRelatedEntityListClick}
        />
    );
};

RelatedEntityStage.propTypes = {
    match: ReactRouterPropTypes.match,
    location: ReactRouterPropTypes.location,
    history: ReactRouterPropTypes.history,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string
};

RelatedEntityStage.defaultProps = {
    match: null,
    location: null,
    history: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null
};

export default RelatedEntityStage;
