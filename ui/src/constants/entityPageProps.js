import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';

export const entityPagePropTypes = {
    entityId: PropTypes.string.isRequired,
    listEntityType1: PropTypes.string,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    query: PropTypes.shape({}),
    sidePanelMode: PropTypes.bool,
    controlResult: PropTypes.shape({}),
    match: ReactRouterPropTypes.match,
    location: ReactRouterPropTypes.location,
    original: PropTypes.shape({})
};

export const entityPageDefaultProps = {
    match: null,
    location: null,
    listEntityType1: null,
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
    query: null,
    sidePanelMode: false,
    controlResult: null,
    original: null
};

export const entityListPropTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    query: PropTypes.shape({}),
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    data: PropTypes.arrayOf(PropTypes.shape({}))
};

export const entityListDefaultprops = {
    className: '',
    selectedRowId: null,
    query: null,
    data: null
};

export const entityComponentPropTypes = {
    id: PropTypes.string.isRequired,
    query: PropTypes.shape({}).isRequired,
    entityListType: PropTypes.string,
    entityContext: PropTypes.shape({})
};

export const entityComponentDefaultProps = {
    entityListType: null,
    contextEntityType: null,
    contextEntityId: null,
    entityContext: {}
};
