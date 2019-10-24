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

// TODO: standardize on entityId and search props from legacy id and query.
export const entityComponentPropTypes = {
    id: PropTypes.string,
    entityId: PropTypes.string,
    query: PropTypes.shape({}),
    search: PropTypes.shape({}),
    entityListType: PropTypes.string,
    entityContext: PropTypes.shape({})
};

export const entityComponentDefaultProps = {
    entityListType: null,
    contextEntityType: null,
    contextEntityId: null,
    entityContext: {},
    query: null,
    search: null,
    id: null,
    entityid: null
};

export const workflowListPropTypes = {
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    sort: PropTypes.string,
    page: PropTypes.number,
    entityContext: PropTypes.shape({})
};

export const workflowListDefaultProps = {
    search: null,
    entityContext: {},
    sort: null,
    page: 1,
    selectedRowId: null
};

export const workflowEntityPropTypes = {
    entityId: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    sort: PropTypes.string,
    page: PropTypes.number
};

export const workflowEntityDefaultProps = {
    entityListType: null,
    entityContext: {},
    search: null,
    sort: null,
    page: 1
};
