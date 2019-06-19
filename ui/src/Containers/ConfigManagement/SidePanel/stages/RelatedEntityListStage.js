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

    return <List entityListType={entityListType2} onRowClick={onRowClick} />;
};

RelatedEntityListStage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.match.isRequired,
    history: ReactRouterPropTypes.match.isRequired,
    entityListType2: PropTypes.string
};

RelatedEntityListStage.defaultProps = {
    entityListType2: null
};

export default RelatedEntityListStage;
