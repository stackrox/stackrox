import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import URLService from 'modules/URLService';

import List from 'Containers/ConfigManagement/EntityList';

const RelatedEntityListStage = ({ match, location, history, entityListType2 }) => {
    function onRowClick(entityId) {
        const urlBuilder = URLService.getURL(match, location).set('entityId2', entityId);
        history.push(urlBuilder.url());
    }

    function onRowLinkClick(entityId, relatedEntityType, relatedEntityId) {
        const urlBuilder = URLService.getURL(match, location).base(
            relatedEntityType,
            relatedEntityId
        );
        history.push(urlBuilder.url());
    }

    return (
        <List
            entityListType={entityListType2}
            onRowClick={onRowClick}
            onRowLinkClick={onRowLinkClick}
        />
    );
};

RelatedEntityListStage.propTypes = {
    match: ReactRouterPropTypes.match,
    location: ReactRouterPropTypes.location,
    history: ReactRouterPropTypes.history,
    entityListType2: PropTypes.string
};

RelatedEntityListStage.defaultProps = {
    match: null,
    location: null,
    history: null,
    entityListType2: null
};

export default RelatedEntityListStage;
