import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';

export const entityPagePropTypes = {
    entityId: PropTypes.string.isRequired,
    listEntityType: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    query: PropTypes.shape({}),
    sidePanelMode: PropTypes.bool,
    controlResult: PropTypes.shape({}),
    match: ReactRouterPropTypes.match,
    location: ReactRouterPropTypes.location
};

export const entityPageDefaultProps = {
    match: null,
    location: null,
    listEntityType: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
    query: null,
    sidePanelMode: false,
    controlResult: null
};
